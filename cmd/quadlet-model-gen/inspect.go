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
	direct      map[lookupFuncArgs]string
	ambiguous   map[lookupFuncArgs]string
	alternative map[lookupFuncArgs]string
}

type lookupFuncArgs struct {
	group string
	key   lookupFuncKey
}

type lookupFuncKey struct {
	name    string
	literal bool
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

func inspectQuadletSourceFileLookupFuncCalls(file *ast.File, declarations declarations, lookupFuncs map[string]lookupFunc) lookupFuncCalls {
	calls := lookupFuncCalls{
		direct:      make(map[lookupFuncArgs]string),
		ambiguous:   make(map[lookupFuncArgs]string),
		alternative: make(map[lookupFuncArgs]string),
	}

	keyConstants := declarations.keyConstants
	groupConstants := declarations.groupConstants

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if len(call.Args) < 2 {
			return true
		}

		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		lookupFuncName := selector.Sel.Name
		if _, isLookupFunc := lookupFuncs[lookupFuncName]; !isLookupFunc {
			return true
		}

		groupConst, ok := call.Args[0].(*ast.Ident)
		if !ok {
			panic(fmt.Sprintf("expected lookup function's 1st argument to be an identifier containing a group name "+
				"(*ast.Ident) but found %T instead. The parser likely needs to be updated", call.Args[0]))
		}

		key := lookupFuncKey{}
		switch arg1 := call.Args[1].(type) {
		case *ast.Ident:
			key.name = arg1.Name
		case *ast.BasicLit:
			if arg1.Kind != token.STRING {
				panic(fmt.Sprintf("expected the 2nd argument of lookup function %s to be a string litreal "+
					"but was %T instead", types.ExprString(selector), call.Args[1]))
			}
			key.name, _ = strconv.Unquote(arg1.Value)
			key.literal = true
		case *ast.IndexExpr: // ignore those and look for alternative lookup functions
		default:
			panic(fmt.Sprintf("expected the 2nd argument of lookup function %s to be an identifier containing a key name "+
				"or a string literal but found %T instead.", types.ExprString(selector), call.Args[1]))
		}

		groupName, okGroup := groupConstants[groupConst.Name]
		keyName, okKey := keyConstants[key.name]

		if okGroup && okKey {
			calls.direct[lookupFuncArgs{
				group: groupName,
				key:   lookupFuncKey{name: keyName},
			}] = lookupFuncName
		} else {
			calls.ambiguous[lookupFuncArgs{
				group: groupConst.Name,
				key:   key,
			}] = lookupFuncName
		}

		return true
	})

	return calls
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
