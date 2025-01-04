package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuadletParser(t *testing.T) {
	t.Parallel()

	lookupFuncs, err := parseUnitFileGo()
	if err != nil {
		t.Fatal(err)
	}

	file, err := os.Open("testdata/v5.3.1/quadlet.go")
	if err != nil {
		t.Fatal(err)
	}

	fieldsByGroup, err := parseQuadletSourceFile(file, lookupFuncs)
	if err != nil {
		t.Fatal(err)
	}

	for _, fields := range fieldsByGroup {
		for _, field := range fields {
			if field.Group == "" {
				t.Errorf("field group is empty: %+v", field)
			}
			if field.Key == "" {
				t.Errorf("field key is empty: %+v", field)
			}
			if field.LookupFunc.Name == "" {
				t.Errorf("field has no LookupFunc: %+v", field)
			}
		}
	}
}

func TestUnitFileParser(t *testing.T) {
	t.Parallel()

	lookupFuncs, err := parseUnitFileGo()
	if err != nil {
		t.Fatal(err)
	}

	expectedLookupFuncs := map[string]lookupFunc{
		"LookupAllStrv":            {Name: "LookupAllStrv", Multiple: true},
		"LookupLastRaw":            {Name: "LookupLastRaw", Multiple: false},
		"LookupLast":               {Name: "LookupLast", Multiple: false},
		"LookupBoolean":            {Name: "LookupBoolean", Multiple: false},
		"LookupBooleanWithDefault": {Name: "LookupBooleanWithDefault", Multiple: false},
		"Lookup":                   {Name: "Lookup", Multiple: false},
		"LookupInt":                {Name: "LookupInt", Multiple: false},
		"LookupLastArgs":           {Name: "LookupLastArgs", Multiple: true},
		"LookupAllKeyVal":          {Name: "LookupAllKeyVal", Multiple: true},
		"LookupUint32":             {Name: "LookupUint32", Multiple: false},
		"LookupUID":                {Name: "LookupUID", Multiple: false},
		"LookupGID":                {Name: "LookupGID", Multiple: false},
		"LookupAll":                {Name: "LookupAll", Multiple: true},
		"LookupAllRaw":             {Name: "LookupAllRaw", Multiple: true},
		"LookupAllArgs":            {Name: "LookupAllArgs", Multiple: true},
	}

	assert.Equal(t, expectedLookupFuncs, lookupFuncs)
}

func parseUnitFileGo() (map[string]lookupFunc, error) {
	parserFile, err := os.Open("testdata/v5.3.1/unitfile.go")
	if err != nil {
		return nil, err
	}

	lookupFuncs, err := parseUnitFileParserSourceFile(parserFile)
	if err != nil {
		return nil, err
	}
	return lookupFuncs, nil
}
