package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

var keyMapByGroup = map[string]string{
	"Container": "supportedContainerKeys",
	"Volume":    "supportedVolumeKeys",
	"Network":   "supportedNetworkKeys",
	"Kube":      "supportedKubeKeys",
	"Image":     "supportedImageKeys",
	"Build":     "supportedBuildKeys",
	"Pod":       "supportedPodKeys",
	"Quadlet":   "supportedQuadletKeys",
}

var groupByKeyMap = reverseMap(keyMapByGroup)

var alternativeLookupMethods = map[string]string{
	"lookupAndAddString":     "Lookup",
	"lookupAndAddAllStrings": "LookupAll",
	"lookupAndAddBoolean":    "LookupBoolean",
}

// Approximate number of elements present in the source files
const (
	nbGroups          = 11
	nbConstants       = 150
	nbKeysPerGroup    = 50
	nbLookupFunctions = 15
)

type sourceFileData struct {
	fieldsByGroup map[string][]field
	lookupFuncs   map[string]lookupFunc
}

type field struct {
	Group      string
	Key        string
	LookupFunc lookupFunc
}

type lookupFunc struct {
	Name     string
	Multiple bool
}

func parseUnitFileParserSourceFile(file *os.File) (map[string]lookupFunc, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file.Name(), nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	lookupFuncs := make(map[string]lookupFunc, nbLookupFunctions)
	for _, decl := range parsed.Decls {
		decl, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasPrefix(decl.Name.Name, "Lookup") {
			continue
		}

		var multiple bool
		for _, field := range decl.Type.Results.List {
			if _, ok := field.Type.(*ast.ArrayType); ok {
				multiple = true
			} else if _, ok := field.Type.(*ast.MapType); ok {
				multiple = true
			}
		}

		lookupFuncs[decl.Name.Name] = lookupFunc{
			Name:     decl.Name.Name,
			Multiple: multiple,
		}
	}

	return lookupFuncs, nil
}

type inspectionResult struct {
	keyConstants      map[string]string
	groupConstants    map[string]string
	supportedKeysMaps map[string][]string
	lookupFuncByArgs  map[lookupFuncArgs]string
}

type lookupFuncArgs struct {
	group string
	key   lookupFuncKey
}

type lookupFuncKey struct {
	value   string
	literal bool
}

func parseQuadletSourceFile(file *os.File, lookupFuncs map[string]lookupFunc) (map[string][]field, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file.Name(), nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	result := inspectQuadletSourceFile(parsed)
	keyConstants := result.keyConstants
	groupConstants := result.groupConstants

	fieldsByGroup := make(map[string][]field, len(groupConstants))

	lookupFuncByGroupKey := make(map[string]map[string]lookupFunc, nbGroups)
	for args, lookupFuncName := range result.lookupFuncByArgs {
		group := groupConstants[args.group]
		if !args.key.literal {
			key := keyConstants[args.key.value]
			lookupFuncByGroupKey[group][key] = lookupFuncs[lookupFuncName]
		} else {
			fieldsByGroup[group] = append(fieldsByGroup[group], field{
				Group:      group,
				Key:        args.key.value,
				LookupFunc: lookupFuncs[lookupFuncName],
			})
		}
	}

	for _, group := range groupConstants {
		keyMapName := keyMapByGroup[group]
		for _, keyConstName := range result.supportedKeysMaps[keyMapName] {
			key := keyConstants[keyConstName]
			fieldsByGroup[group] = append(fieldsByGroup[group], field{
				Group:      group,
				Key:        key,
				LookupFunc: lookupFuncByGroupKey[group][key],
			})
		}
	}

	return fieldsByGroup, nil
}

func inspectQuadletSourceFile(file *ast.File) inspectionResult {
	result := inspectionResult{
		groupConstants:    make(map[string]string, nbGroups),
		keyConstants:      make(map[string]string, nbConstants),
		supportedKeysMaps: make(map[string][]string, len(groupByKeyMap)),
		lookupFuncByArgs:  make(map[lookupFuncArgs]string),
	}

	for _, decl := range file.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if decl.Tok == token.CONST {
			for _, spec := range decl.Specs {
				valueSpec, name := mustGetValueSpecName(spec)
				if isKeyNameConst(name) {
					value := mustExtractConstantValue(valueSpec, name)
					result.keyConstants[name] = value
				} else if isGroupNameConst(name) {
					value := mustExtractConstantValue(valueSpec, name)
					result.groupConstants[name] = value
				}
			}
		}

		if decl.Tok == token.VAR {
			for _, spec := range decl.Specs {
				valueSpec, name := mustGetValueSpecName(spec)
				if _, ok := groupByKeyMap[name]; !ok {
					continue
				}
				keys := mustExtractSupportedKeysFromMap(valueSpec, name)
				result.supportedKeysMaps[name] = keys
			}
		}
	}

	return result
}

func isKeyNameConst(name string) bool {
	return strings.HasPrefix(name, "Key")
}

func isGroupNameConst(name string) bool {
	return strings.HasSuffix(name, "Group") &&
		!strings.HasPrefix(name, "X") &&
		!strings.HasPrefix(name, "Key")
}

func mustExtractSupportedKeysFromMap(spec *ast.ValueSpec, name string) []string {
	if len(spec.Values) != 1 {
		panic(fmt.Sprintf("quadlet.go should only have constants that have a single value. "+
			"Spec %s has %d values. The parser likely needs to be updated", name, len(spec.Values)))
	}

	value, ok := spec.Values[0].(*ast.CompositeLit)
	_, isMap := value.Type.(*ast.MapType)
	if !ok || !isMap {
		panic(fmt.Sprintf("quadlet.go should only have variables defining maps of supported keys that have map literal values of type *ast.CompositeLit. "+
			"Spec %s is of type %T with values of type %T. The parser likely needs to be updated", name, value.Type, spec.Values[0]))
	}

	keys := make([]string, 0, len(value.Elts))
	for _, elt := range value.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			panic(fmt.Sprintf("quadlet.go should only have key-value composite literals should be of type *ast.KeyValueExpr. "+
				"Spec %s has composite literal of type %T. The parser likely needs to be updated", name, elt))
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			panic(fmt.Sprintf("quadlet.go should only have keys of key-value composite literals that are of type *ast.Ident. "+
				"Spec %s has key of type %T. The parser likely needs to be updated", name, kv.Key))
		}

		keys = append(keys, key.Name)
	}

	return keys
}

func mustExtractConstantValue(spec *ast.ValueSpec, name string) string {

	if len(spec.Values) != 1 {
		panic(fmt.Sprintf("quadlet.go should only have constants that have a single value. "+
			"Spec %s has %d values. The parser likely needs to be updated", name, len(spec.Values)))
	}

	value, ok := spec.Values[0].(*ast.BasicLit)
	if !ok || value.Kind != token.STRING {
		panic(fmt.Sprintf("quadlet.go should only have constants that have string literal values. "+
			"Spec %s is of kind %s and of type %T. The parser likely needs to be updated", name, value.Kind.String(), spec.Values[0]))
	}

	unquoted, err := strconv.Unquote(value.Value)
	if err != nil {
		panic(fmt.Sprintf("syntax error while unquoting value %s of valueSpec %s", value.Value, name))
	}
	return unquoted
}

func mustGetValueSpecName(spec ast.Spec) (*ast.ValueSpec, string) {
	valueSpec, ok := spec.(*ast.ValueSpec)
	if !ok {
		panic(fmt.Sprintf("quadlet.go should only have constants of type *ast.ValueSpec. "+
			"Spec %s is of type %T. The parser likely needs to be updated", valueSpec.Names, valueSpec))
	}

	if len(valueSpec.Names) != 1 {
		panic(fmt.Sprintf("quadlet.go should only have constants that have a single name. "+
			"Spec %s has %d names. The parser likely needs to be updated", valueSpec.Names, len(valueSpec.Names)))
	}

	name := valueSpec.Names[0].Name
	return valueSpec, name
}

// TODO: This is bad. Refactor.
//
//nolint:all
func parseQuadletSourceFile2(file *os.File, lookupFuncs map[string]lookupFunc) (map[string][]field, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file.Name(), nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	constants := make(map[string]string, nbConstants)
	groups := make(map[string][]field, nbGroups) // The number of groups declared in the file
	groupNameByGroupVarName := make(map[string]string, len(groups))
	keyNameByKeyVarName := make(map[string]string, nbKeysPerGroup)
	for _, decl := range parsed.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok || (decl.Tok != token.VAR && decl.Tok != token.CONST) {
			continue
		}

		if decl.Tok == token.CONST {
			for _, spec := range decl.Specs {
				spec, _ := spec.(*ast.ValueSpec)
				name := spec.Names[0].Name
				if len(spec.Values) != 1 {
					continue
				}
				value, _ := spec.Values[0].(*ast.BasicLit)
				if value.Kind == token.STRING {
					constants[name] = strings.ReplaceAll(value.Value, "\"", "")
				}
			}
		}

		// Extract the group names from the ...Group const values like ContainerGroup, NetworkGroup, etc.
		for _, spec := range decl.Specs {
			spec, _ := spec.(*ast.ValueSpec)
			if group, groupVar, ok := getGroupName(spec); ok {
				groups[group] = make([]field, 0, nbKeysPerGroup)
				groupNameByGroupVarName[groupVar] = group
			}
		}

		// Extract the key variable names from the Key... const values like KeyImage, KeyExec, etc.
		for _, spec := range decl.Specs {
			spec, _ := spec.(*ast.ValueSpec)
			if keyVar, keyName, ok := getKeyVarName(spec); ok {
				keyNameByKeyVarName[keyVar] = keyName
			}
		}
	}

	for _, decl := range parsed.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok || (decl.Tok != token.VAR && decl.Tok != token.CONST) {
			continue
		}

		// For every group, map the key variable names with the key names
		for _, spec := range decl.Specs {
			spec, _ := spec.(*ast.ValueSpec)
			if group, fields, ok := getGroupFields(spec, keyNameByKeyVarName); ok {
				groups[group] = fields
			}
		}
	}

	// Map to hold CallExpr and their enclosing functions
	parentFunctions := make(map[*ast.CallExpr]*ast.FuncDecl)
	callExprs := make(map[string][]*ast.CallExpr)

	var currentFunc *ast.FuncDecl
	ast.Inspect(parsed, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			currentFunc = node
		case *ast.CallExpr:
			if currentFunc != nil {
				var callName string
				if ident, ok := node.Fun.(*ast.Ident); ok {
					callName = ident.Name
				} else if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
					callName = sel.Sel.Name
				}
				parentFunctions[node] = currentFunc

				if _, ok := callExprs[callName]; !ok {
					callExprs[callName] = make([]*ast.CallExpr, 0)
				}
				callExprs[callName] = append(callExprs[callName], node)
			}
		}
		return true
	})

	ast.Inspect(parsed, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			switch fun := callExpr.Fun.(type) {
			case *ast.SelectorExpr:
				_, ok := fun.X.(*ast.Ident)
				if lookupFunc, lookupFuncFound := lookupFuncs[fun.Sel.Name]; ok && lookupFuncFound {
					args := callExpr.Args
					var group, key string
					var okGroup, okKey bool
					if groupIdent, ok := args[0].(*ast.Ident); ok {
						group, okGroup = groupNameByGroupVarName[groupIdent.Name]
					}

					if keyIdent, ok := args[1].(*ast.Ident); ok {
						key, okKey = keyNameByKeyVarName[keyIdent.Name]
						if keyFromConstants, ok := constants[keyIdent.Name]; !okKey && ok {
							key = keyFromConstants
							okKey = true
						}
					} else if basicLit, ok := args[1].(*ast.BasicLit); ok {
						key = strings.ReplaceAll(basicLit.Value, "\"", "")
						okKey = true
					}

					if okGroup && okKey {
						var found bool
						for i := range groups[group] {
							if groups[group][i].Key == key {
								groups[group][i].LookupFunc = lookupFunc
								found = true

								break
							}
						}

						if !found {
							groups[group] = append(groups[group], field{
								Group:      group,
								Key:        key,
								LookupFunc: lookupFunc,
							})
						}
					} else if !okGroup && okKey {
						for _, fields := range groups {
							for i := range fields {
								if fields[i].Key == key {
									fields[i].LookupFunc = lookupFunc

									break
								}
							}
						}
					}
				}
			case *ast.Ident:
				if lookupFuncName, ok := alternativeLookupMethods[fun.Name]; ok {
					lookupFunc := lookupFuncs[lookupFuncName]
					for _, expr := range callExprs[fun.Name] {
						if expr.Pos() == fun.Pos() {
							groupVarName := expr.Args[1].(*ast.Ident).Name
							group := groupNameByGroupVarName[groupVarName]
							keysVarName := expr.Args[2].(*ast.Ident).Name
							ast.Inspect(parentFunctions[expr], func(n ast.Node) bool {
								if assign, ok := n.(*ast.AssignStmt); ok {
									if len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
										if ident, ok := assign.Lhs[0].(*ast.Ident); ok && ident.Name == keysVarName {
											if composite, ok := assign.Rhs[0].(*ast.CompositeLit); ok {
												if _, ok := composite.Type.(*ast.MapType); ok {
													for _, elt := range composite.Elts {
														kv := elt.(*ast.KeyValueExpr)
														key := keyNameByKeyVarName[kv.Key.(*ast.Ident).Name]
														for i := range groups[group] {
															if groups[group][i].Key == key {
																groups[group][i].LookupFunc = lookupFunc
															}
														}
													}
												}
											}
										}
									}
								}
								return true
							})
							break
						}
					}
				}
			}
			return false
		}
		return true
	})

	for group, fields := range groups {
		// Easier this way...
		for i, field := range fields {
			if strings.HasPrefix(field.Key, "Health") {
				groups[group][i].LookupFunc = lookupFuncs["Lookup"]
			}
		}
	}

	return groups, nil
}

// getKeyVarName extracts the key name and the variable containing the key name from the constant declaration
// passed to the function
func getKeyVarName(spec *ast.ValueSpec) (string, string, bool) {
	keyVar := spec.Names[0].Name
	if strings.HasPrefix(keyVar, "Key") && len(spec.Values) == 1 {
		value, ok := spec.Values[0].(*ast.BasicLit)
		if !ok || value.Kind != token.STRING {
			return "", "", false
		}

		return keyVar, strings.ReplaceAll(value.Value, "\"", ""), true
	}

	return "", "", false
}

// getGroupFields extracts the group and its fields from the map variables listing the supported fields for each group
func getGroupFields(spec *ast.ValueSpec, keyNameByKeyVarName map[string]string) (string, []field, bool) {
	group, ok := groupByKeyMap[spec.Names[0].Name]
	if !ok {
		return "", nil, false
	}

	value, _ := spec.Values[0].(*ast.CompositeLit)
	fields := make([]field, 0, len(value.Elts))
	for _, elt := range value.Elts {
		kv, _ := elt.(*ast.KeyValueExpr)
		keyVarName, _ := kv.Key.(*ast.Ident)
		if keyName, ok := keyNameByKeyVarName[keyVarName.Name]; ok {
			fields = append(fields, field{Group: group, Key: keyName})
		}
	}

	return group, fields, true
}

// getGroupName extracts the group name and the variable containing the group name from the constant declaration
// passed to the function
func getGroupName(spec *ast.ValueSpec) (string, string, bool) {
	groupVarName := spec.Names[0].Name
	if strings.HasSuffix(groupVarName, "Group") &&
		!strings.HasPrefix(groupVarName, "X") &&
		!strings.HasPrefix(groupVarName, "Key") &&
		len(spec.Values) == 1 {
		value, _ := spec.Values[0].(*ast.BasicLit)
		if value.Kind != token.STRING {
			return "", groupVarName, false
		}

		return strings.ReplaceAll(value.Value, "\"", ""), groupVarName, true
	}

	return "", groupVarName, false
}
