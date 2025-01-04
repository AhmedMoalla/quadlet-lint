package main

import (
	"os"
	"testing"
)

func TestQuadletParser(t *testing.T) {
	t.Parallel()

	file, err := os.Open("testdata/quadlet.go")
	if err != nil {
		t.Fatal(err)
	}

	parserFile, err := os.Open("testdata/unitfile.go")
	if err != nil {
		t.Fatal(err)
	}

	lookupFuncs, err := parseUnitFileParserSourceFile(parserFile)
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
