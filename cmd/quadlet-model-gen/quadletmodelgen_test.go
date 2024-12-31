//go:generate ./quadletmodelgen_ref.sh ".generated-ref"
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	generatedRefDirName = ".generated-ref"
)

func TestQuadletModelGen(t *testing.T) {
	generatedRefDir, err := os.Open(generatedRefDirName)
	if err != nil && os.IsNotExist(err) {
		t.Fatal(errors.Join(err, errors.New("'go generate' was not run before starting the test")))
	}

	podmanVersion, err := getPodmanVersionFromGeneratedComment()
	if err != nil {
		t.Fatal(err)
	}

	runLinter(podmanVersion)
	defer os.RemoveAll(generatedDirName)
	generatedDir, err := os.Open(generatedRefDirName)
	if err != nil && os.IsNotExist(err) {
		t.Fatal(err)
	}

	compareFiles(generatedRefDir, generatedDir)
}

func compareFiles(generatedRefDir *os.File, generatedDir *os.File) {
	// TODO: Do comparison
}

func getPodmanVersionFromGeneratedComment() (string, error) {
	file, err := os.Open(generatedRefDirName + "/groups.go")
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(file)
	headComment, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	commentSections := strings.SplitN(headComment, ";", 3)
	if len(commentSections) != 3 {
		return "", fmt.Errorf("could not find Podman version. invalid head comment: '%s'", headComment)
	}

	podmanVersionKV := strings.SplitN(commentSections[1], "=", 2)
	if len(podmanVersionKV) != 2 {
		return "", fmt.Errorf("could not find Podman version. invalid head comment: '%s'", commentSections[1])
	}

	return podmanVersionKV[1], nil
}
