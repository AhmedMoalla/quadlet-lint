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

	fields, err := parseQuadletSourceFile(file, nil)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%#v\n", fields)
}
