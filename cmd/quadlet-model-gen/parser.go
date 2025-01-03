package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
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

func parseQuadletSourceFile(file *os.File, lookupFuncs map[string]lookupFunc) (map[string][]field, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file.Name(), nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	declarations := inspectQuadletSourceFileDeclarations(parsed)
	otherConstants := declarations.otherConstants
	keyConstants := declarations.keyConstants
	groupConstants := declarations.groupConstants
	supportedKeysMaps := declarations.supportedKeysMaps

	lookupCalls := inspectQuadletSourceFileLookupFuncCalls(parsed, declarations, lookupFuncs)

	fieldsByGroup := make(map[string][]field, len(groupConstants))

	lookupFuncByGroupKey := make(map[string]map[string]lookupFunc, nbGroups)
	for _, group := range groupConstants {
		lookupFuncByGroupKey[group] = make(map[string]lookupFunc, nbKeysPerGroup)
	}

	for args, funcName := range lookupCalls.direct {
		lookupFuncByGroupKey[args.group][args.key.name] = lookupFuncs[funcName]
	}

	for args, funcName := range lookupCalls.ambiguous {
		if group, ok := groupConstants[args.group]; ok && args.key.literal {
			fieldsByGroup[group] = append(fieldsByGroup[group], field{
				Group:      group,
				Key:        args.key.name,
				LookupFunc: lookupFuncs[funcName],
			})
			continue
		} else if ok && !args.key.literal { // Key constant is not prefixed with Key
			fieldsByGroup[group] = append(fieldsByGroup[group], field{
				Group:      group,
				Key:        otherConstants[args.key.name],
				LookupFunc: lookupFuncs[funcName],
			})
			continue
		}

		for mapName, keyVarNames := range supportedKeysMaps {
			group := groupByKeyMap[mapName]
			for _, keyVarName := range keyVarNames {
				if keyVarName == args.key.name {
					key := keyConstants[args.key.name]
					lookupFuncByGroupKey[group][key] = lookupFuncs[funcName]
					break
				}
			}
		}
	}

	for _, group := range groupConstants {
		keyMapName := keyMapByGroup[group]
		for _, keyConstName := range supportedKeysMaps[keyMapName] {
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
