package main

import (
	"github.com/AhmedMoalla/quadlet-lint/pkg/generated"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
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

type quadletSourceFileData struct {
	fieldsByGroup map[string][]string
}

func parseQuadletSourceFile(file *os.File) (quadletSourceFileData, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file.Name(), nil, parser.SkipObjectResolution)
	if err != nil {
		return quadletSourceFileData{}, err
	}

	groups := make(map[string][]string, 11) // The number of groups declared in the file
	keyNameByKeyVarName := make(map[string]string, 50)
	for _, decl := range parsed.Decls {
		decl, ok := decl.(*ast.GenDecl)
		if !ok || (decl.Tok != token.VAR && decl.Tok != token.CONST) {
			continue
		}

		for _, spec := range decl.Specs {
			spec := spec.(*ast.ValueSpec)
			if group, ok := getGroupName(spec); ok {
				groups[group] = make([]string, 0, 50)
			}
		}

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
		for _, spec := range decl.Specs {
			spec := spec.(*ast.ValueSpec)
			if group, keyVars, ok := getGroupKeys(spec, keyNameByKeyVarName); ok {
				groups[group] = keyVars
			}
		}
	}

	for group := range groups {
		if fields, ok := generated.AdditionalFields[group]; ok {
			groups[group] = append(groups[group], fields...)
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

func getGroupKeys(spec *ast.ValueSpec, keyNameByKeyVarName map[string]string) (group string, keysVarNames []string, isKeyMap bool) {
	group, ok := groupByKeyMap[spec.Names[0].Name]
	if !ok {
		return "", nil, false
	}

	value := spec.Values[0].(*ast.CompositeLit)
	keysVarNames = make([]string, 0, len(value.Elts))
	for _, elt := range value.Elts {
		kv := elt.(*ast.KeyValueExpr)
		keyVarName := kv.Key.(*ast.Ident)
		if keyName, ok := keyNameByKeyVarName[keyVarName.Name]; ok {
			keysVarNames = append(keysVarNames, keyName)
		}
	}

	return group, keysVarNames, true
}

func getGroupName(spec *ast.ValueSpec) (string, bool) {
	groupName := spec.Names[0].Name
	if strings.HasSuffix(groupName, "Group") &&
		!strings.HasPrefix(groupName, "X") &&
		!strings.HasPrefix(groupName, "Key") &&
		len(spec.Values) == 1 {
		value := spec.Values[0].(*ast.BasicLit)
		if value.Kind != token.STRING {
			return "", false
		}

		return strings.Replace(value.Value, "\"", "", 2), true
	}

	return "", false
}
