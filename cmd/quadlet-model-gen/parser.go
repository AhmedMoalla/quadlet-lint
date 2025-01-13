package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/AhmedMoalla/quadlet-lint/pkg/utils"
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

var groupByKeyMap = utils.ReverseMap(keyMapByGroup)

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
	calls := inspectQuadletSourceFileLookupCalls(parsed, declarations, lookupFuncs)
	lookupFuncByGroupKey, additionalFields := parseLookupCalls(declarations, calls)

	groupConstants := declarations.groupConstants
	fieldsByGroup := make(map[string][]field, len(groupConstants))
	for _, group := range groupConstants {
		keyMapName := keyMapByGroup[group]
		for _, keyConstName := range declarations.supportedKeysMaps[keyMapName] {
			key := declarations.keyConstants[keyConstName]
			fieldsByGroup[group] = append(fieldsByGroup[group], field{
				Group:      group,
				Key:        key,
				LookupFunc: lookupFuncByGroupKey[group][key],
			})
		}
	}

	// Easier this way...
	fieldsByGroup = utils.MergeMaps(fieldsByGroup, additionalFields)
	for group, fields := range fieldsByGroup {
		for i, field := range fields {
			if strings.HasPrefix(field.Key, "Health") {
				fieldsByGroup[group][i].LookupFunc = lookupFuncs["Lookup"]
			}
		}
	}

	return fieldsByGroup, nil
}

func parseLookupCalls(
	declarations declarations,
	calls lookupFuncCalls,
) (map[string]map[string]lookupFunc, map[string][]field) {
	otherConstants := declarations.otherConstants
	groupConstants := declarations.groupConstants
	keyConstants := declarations.keyConstants
	supportedKeysMaps := declarations.supportedKeysMaps

	lookupFuncByGroupKey := make(map[string]map[string]lookupFunc, nbGroups)
	for _, group := range groupConstants {
		lookupFuncByGroupKey[group] = make(map[string]lookupFunc, nbKeysPerGroup)
	}

	for args, lookupFunc := range calls.direct {
		lookupFuncByGroupKey[args.group][args.key.name] = lookupFunc
	}

	for args, lookupFunc := range calls.alternative {
		lookupFuncByGroupKey[args.group][args.key.name] = lookupFunc
	}

	additionalFields := make(map[string][]field, len(groupConstants))
	for args, lookupFunc := range calls.ambiguous {
		if group, ok := groupConstants[args.group]; ok {
			var key string
			if args.key.literal {
				key = args.key.name
			} else {
				key = otherConstants[args.key.name]
			}
			additionalFields[group] = append(additionalFields[group], field{
				Group:      group,
				Key:        key,
				LookupFunc: lookupFunc,
			})
			continue
		}

		for mapName, keyVarNames := range supportedKeysMaps {
			group := groupByKeyMap[mapName]
			for _, keyVarName := range keyVarNames {
				if keyVarName == args.key.name {
					key := keyConstants[args.key.name]
					lookupFuncByGroupKey[group][key] = lookupFunc
					break
				}
			}
		}
	}

	return lookupFuncByGroupKey, additionalFields
}
