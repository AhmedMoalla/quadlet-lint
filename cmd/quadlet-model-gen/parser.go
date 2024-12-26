package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model"
)

var groupByKeyMap = map[string]string{
	"supportedContainerKeys": "Container",
	"supportedVolumeKeys":    "Volume",
	"supportedNetworkKeys":   "Network",
	"supportedKubeKeys":      "Kube",
	"supportedImageKeys":     "Image",
	"supportedBuildKeys":     "Build",
	"supportedPodKeys":       "Pod",
	"supportedQuadletKeys":   "Quadlet",
}

var alternativeLookupMethods = map[string]string{
	"lookupAndAddString":     "Lookup",
	"lookupAndAddAllStrings": "LookupAll",
	"lookupAndAddBoolean":    "LookupBoolean",
}

type quadletSourceFileData struct {
	fieldsByGroup map[string][]model.Field
}

// TODO: Generate Field instances for keys used as string literals like Service.KillMode
// - Extract all Lookup function calls from the quadlet.go file
// - Get all the calls where the key parameter (second arg) is a string literal
// - For every call extract the group and key and add it as a field for the group
// - If the call is a LookupFunc that is flagged as multiple, then the field should be flagged as multiple too
// TODO: Generate LookupFunc instances
// - Extract all method declarations of UnitFile that start with 'Lookup' from the unitfile.go file
// - For every function found, generate a LookupFunc instance
// - If the function's return type is a []string, then it should be flagged as multiple
// TODO: Write tests for generated code
func parseQuadletSourceFile(file *os.File) (quadletSourceFileData, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file.Name(), nil, parser.SkipObjectResolution)
	if err != nil {
		return quadletSourceFileData{}, err
	}

	constants := make(map[string]string, 150)
	groups := make(map[string][]model.Field, 11) // The number of groups declared in the file
	groupNameByGroupVarName := make(map[string]string, len(groups))
	keyNameByKeyVarName := make(map[string]string, 50)
	for _, decl := range parsed.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok || (decl.Tok != token.VAR && decl.Tok != token.CONST) {
			continue
		}

		if decl.Tok == token.CONST {
			for _, spec := range decl.Specs {
				spec := spec.(*ast.ValueSpec)
				name := spec.Names[0].Name
				if len(spec.Values) != 1 {
					continue
				}
				value := spec.Values[0].(*ast.BasicLit)
				if value.Kind == token.STRING {
					constants[name] = strings.ReplaceAll(value.Value, "\"", "")
				}
			}
		}

		// Extract the group names from the ...Group const values like ContainerGroup, NetworkGroup, etc.
		for _, spec := range decl.Specs {
			spec := spec.(*ast.ValueSpec)
			if group, groupVar, ok := getGroupName(spec); ok {
				groups[group] = make([]model.Field, 0, 50)
				groupNameByGroupVarName[groupVar] = group
			}
		}

		// Extract the key variable names from the Key... const values like KeyImage, KeyExec, etc.
		for _, spec := range decl.Specs {
			spec := spec.(*ast.ValueSpec)
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
			spec := spec.(*ast.ValueSpec)
			if group, fields, ok := getGroupFields(spec, keyNameByKeyVarName); ok {
				groups[group] = fields
			}
		}
	}

	// Map to hold CallExpr and their enclosing functions
	parentFunctions := make(map[*ast.CallExpr]*ast.FuncDecl)
	funcDecls := make(map[string][]*ast.FuncDecl)
	callExprs := make(map[string][]*ast.CallExpr)

	// Traverse the AST
	var currentFunc *ast.FuncDecl
	ast.Inspect(parsed, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			// Track the current function declaration
			currentFunc = node
		case *ast.CallExpr:
			// Associate the CallExpr with the current function
			if currentFunc != nil {
				var callName string
				if ident, ok := node.Fun.(*ast.Ident); ok {
					callName = ident.Name
				} else if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
					callName = sel.Sel.Name
				}
				funcName := currentFunc.Name.Name

				parentFunctions[node] = currentFunc
				if _, ok := funcDecls[funcName]; !ok {
					funcDecls[funcName] = make([]*ast.FuncDecl, 0)
				}
				funcDecls[funcName] = append(funcDecls[funcName], currentFunc)

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
				if lookupFunc, lookupFuncFound := model.AllLookupFuncs[fun.Sel.Name]; ok && lookupFuncFound {
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
						for i := range groups[group] {
							if groups[group][i].Key == key {
								groups[group][i].LookupFunc = lookupFunc
								break
							}
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
					lookupFunc := model.AllLookupFuncs[lookupFuncName]
					for _, expr := range callExprs[fun.Name] {
						if expr.Pos() == fun.Pos() {
							groupVarName := expr.Args[1].(*ast.Ident).Name
							group := groupNameByGroupVarName[groupVarName]
							keysVarName := expr.Args[2].(*ast.Ident).Name
							ast.Inspect(parentFunctions[expr], func(n ast.Node) bool {
								switch assign := n.(type) {
								case *ast.AssignStmt:
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
		}
		return true
	})

	for group, fields := range groups {
		if fields, ok := model.AdditionalFields[group]; ok {
			for _, field := range fields {
				groups[group] = append(groups[group], model.Field{Group: group, Key: field})
			}
		}

		// Easier this way...
		for i, field := range fields {
			if strings.HasPrefix(field.Key, "Health") {
				groups[group][i].LookupFunc = model.Lookup
			}
		}
	}

	return quadletSourceFileData{fieldsByGroup: groups}, nil
}

func getKeyVarName(spec *ast.ValueSpec) (keyVar string, keyName string, isKeyVar bool) {
	keyVar = spec.Names[0].Name
	if strings.HasPrefix(keyVar, "Key") && len(spec.Values) == 1 {
		value := spec.Values[0].(*ast.BasicLit)
		if value.Kind != token.STRING {
			return "", "", false
		}

		return keyVar, strings.Replace(value.Value, "\"", "", 2), true
	}

	return "", "", false
}

func getGroupFields(spec *ast.ValueSpec, keyNameByKeyVarName map[string]string) (group string, fields []model.Field, isKeyMap bool) {
	group, ok := groupByKeyMap[spec.Names[0].Name]
	if !ok {
		return "", nil, false
	}

	value := spec.Values[0].(*ast.CompositeLit)
	fields = make([]model.Field, 0, len(value.Elts))
	for _, elt := range value.Elts {
		kv := elt.(*ast.KeyValueExpr)
		keyVarName := kv.Key.(*ast.Ident)
		if keyName, ok := keyNameByKeyVarName[keyVarName.Name]; ok {
			fields = append(fields, model.Field{Group: group, Key: keyName})
		}
	}

	return group, fields, true
}

func getGroupName(spec *ast.ValueSpec) (groupName string, groupVarName string, isGroupDecl bool) {
	groupVarName = spec.Names[0].Name
	if strings.HasSuffix(groupVarName, "Group") &&
		!strings.HasPrefix(groupVarName, "X") &&
		!strings.HasPrefix(groupVarName, "Key") &&
		len(spec.Values) == 1 {
		value := spec.Values[0].(*ast.BasicLit)
		if value.Kind != token.STRING {
			return "", groupVarName, false
		}

		return strings.Replace(value.Value, "\"", "", 2), groupVarName, true
	}

	return "", groupVarName, false
}
