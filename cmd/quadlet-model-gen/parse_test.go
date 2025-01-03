package main

import (
	"fmt"
	"testing"
)

func TestQuadletParser(t *testing.T) {
	file, err := downloadSourceFileFromGithub(quadletFileLocation, "v5.3.1")
	if err != nil {
		t.Fatal(err)
	}

	parserFile, err := downloadSourceFileFromGithub(unitfileParserFileLocation, "v5.3.1")
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

	for group, fields := range fieldsByGroup {
		fmt.Println(group)
		for _, field := range fields {
			if field.LookupFunc.Name == "" {
				fmt.Printf("\t%#v\n", field)
			}
		}
	}
}
