package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

type declarations struct {
	otherConstants    map[string]string
	keyConstants      map[string]string
	groupConstants    map[string]string
	supportedKeysMaps map[string][]string
}

type lookupFuncCalls struct {
	direct      map[lookupFuncArgs]lookupFunc
	ambiguous   map[lookupFuncArgs]lookupFunc
	alternative map[lookupFuncArgs]lookupFunc
}

type lookupFuncArgs struct {
	group string
	key   lookupFuncKey
}

type lookupFuncKey struct {
	name    string
	literal bool
}

var alternativeLookupFuncs = map[string]string{
	"lookupAndAddString":     "Lookup",
	"lookupAndAddAllStrings": "LookupAll",
	"lookupAndAddBoolean":    "LookupBoolean",
}

func inspectQuadletSourceFileDeclarations(file *ast.File) declarations {
	result := declarations{
		otherConstants:    make(map[string]string),
		groupConstants:    make(map[string]string, nbGroups),
		keyConstants:      make(map[string]string, nbConstants),
		supportedKeysMaps: make(map[string][]string, len(groupByKeyMap)),
	}

	for _, decl := range file.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if decl.Tok == token.CONST {
			for _, spec := range decl.Specs {
				valueSpec, name := mustGetValueSpecName(spec)
				value := mustExtractConstantValue(valueSpec, name)
				if isKeyNameConst(name) {
					result.keyConstants[name] = value
				} else if isGroupNameConst(name) {
					result.groupConstants[name] = value
				} else {
					result.otherConstants[name] = value
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

func inspectQuadletSourceFileLookupCalls(file *ast.File, declarations declarations, lookupFuncs map[string]lookupFunc) lookupFuncCalls {
	calls := lookupFuncCalls{
		direct:      make(map[lookupFuncArgs]lookupFunc),
		ambiguous:   make(map[lookupFuncArgs]lookupFunc),
		alternative: make(map[lookupFuncArgs]lookupFunc),
	}

	keyConstants := declarations.keyConstants
	groupConstants := declarations.groupConstants

	var parentFunc *ast.FuncDecl
	ast.Inspect(file, func(n ast.Node) bool {
		if decl, ok := n.(*ast.FuncDecl); ok {
			parentFunc = decl
			return true
		}

		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}

		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			ident, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}

			_, ok = alternativeLookupFuncs[ident.Name]
			if !ok {
				return true
			}

			groupArgIndex := 1
			keyArgIndex := 2
			groupConstName, keysMap := mustGetLookupFuncArgs(call, groupArgIndex, keyArgIndex)

			keyConstNames := mustGetMapKeyFromVariableDefinition(parentFunc.Body.List, keysMap.name)
			for _, keyConstName := range keyConstNames {
				calls.alternative[lookupFuncArgs{
					group: groupConstants[groupConstName],
					key:   lookupFuncKey{name: keyConstants[keyConstName]},
				}] = lookupFuncs[alternativeLookupFuncs[ident.Name]]
			}

			return true
		}

		lookupFunc, isLookupFunc := lookupFuncs[selector.Sel.Name]
		if !isLookupFunc {
			return true
		}

		groupArgIndex := 0
		keyArgIndex := 1
		groupConstName, key := mustGetLookupFuncArgs(call, groupArgIndex, keyArgIndex)

		groupName, okGroup := groupConstants[groupConstName]
		keyName, okKey := keyConstants[key.name]

		if okGroup && okKey {
			calls.direct[lookupFuncArgs{
				group: groupName,
				key:   lookupFuncKey{name: keyName},
			}] = lookupFunc
		} else {
			calls.ambiguous[lookupFuncArgs{
				group: groupConstName,
				key:   key,
			}] = lookupFunc
		}

		return true
	})

	return calls
}

func mustGetMapKeyFromVariableDefinition(statements []ast.Stmt, name string) []string {
	for _, statement := range statements {
		statement, ok := statement.(*ast.AssignStmt)
		if !ok || statement.Tok != token.ASSIGN && statement.Tok != token.DEFINE {
			continue
		}

		var value *ast.CompositeLit
		for i, variable := range statement.Lhs {
			ident, ok := variable.(*ast.Ident)
			if !ok {
				continue
			}

			if ident.Name != name {
				continue
			}

			variableValue := statement.Rhs[i]
			value, ok = variableValue.(*ast.CompositeLit)
			if !ok {
				panic(fmt.Sprintf("expected variable %s to be a composite literal but found type %T instead", name, variableValue))
			}

			break
		}

		if value == nil {
			continue
		}

		_, isMap := value.Type.(*ast.MapType)
		if !isMap {
			panic(fmt.Sprintf("expected variable %s to be a map but found type %T instead", name, value.Type))
		}

		keys := make([]string, 0, len(value.Elts))
		for _, elt := range value.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				panic(fmt.Sprintf("expected elements of composite literal of variable %s to be of of type "+
					"*ast.KeyValueExpr but found type %T instead", name, elt))
			}

			keyConst, ok := kv.Key.(*ast.Ident)
			if !ok {
				panic(fmt.Sprintf("expected key of key-value element of composite literal of variable %s to be of of type "+
					"*ast.Ident but found type %T instead", name, kv.Key))
			}

			keys = append(keys, keyConst.Name)
		}

		return keys
	}

	return nil
}

func mustGetLookupFuncArgs(call *ast.CallExpr, groupArgIndex, keyArgIndex int) (string, lookupFuncKey) {
	groupConst, ok := call.Args[groupArgIndex].(*ast.Ident)
	if !ok {
		panic(fmt.Sprintf("expected lookup function's 1st argument to be an identifier containing a group name "+
			"(*ast.Ident) but found %T instead. The parser likely needs to be updated", call.Args[0]))
	}

	key := lookupFuncKey{}
	switch arg1 := call.Args[keyArgIndex].(type) {
	case *ast.Ident:
		key.name = arg1.Name
	case *ast.BasicLit:
		if arg1.Kind != token.STRING {
			panic(fmt.Sprintf("expected the 2nd argument of lookup function %s to be a string litreal "+
				"but was %T instead. The parser likely needs to be updated", types.ExprString(call.Fun), call.Args[1]))
		}
		key.name, _ = strconv.Unquote(arg1.Value)
		key.literal = true
	case *ast.IndexExpr: // ignore those and look for alternative lookup functions
	default:
		panic(fmt.Sprintf("expected the 2nd argument of lookup function %s to be an identifier containing a key name "+
			"or a string literal but found %T instead. The parser likely needs to be updated", types.ExprString(call.Fun), call.Args[1]))
	}
	return groupConst.Name, key
}

func isKeyNameConst(name string) bool {
	return strings.HasPrefix(name, "Key")
}

func isGroupNameConst(name string) bool {
	return strings.HasSuffix(name, "Group") &&
		!strings.HasPrefix(name, "X") &&
		!isKeyNameConst(name)
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
