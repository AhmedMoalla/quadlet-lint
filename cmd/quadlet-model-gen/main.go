package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	podmanVersionFlag          = "podman-version"
	podmanVersionEnvKey        = "PODMAN_VERSION"
	quadletFileLocation        = "https://raw.githubusercontent.com/containers/podman/refs/tags/%s/pkg/systemd/quadlet/quadlet.go"
	unitfileParserFileLocation = "https://raw.githubusercontent.com/containers/podman/refs/tags/%s/pkg/systemd/parser/unitfile.go"
	baseModelPackageName       = "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
)

var (
	podmanVersion = flag.String(podmanVersionFlag, "", "Podman's tag used to download the source file for code generation")
)

func main() {
	flag.Parse()

	podmanVersion := getPodmanVersion(*podmanVersion)
	unitfileParserFile, err := downloadSourceFileFromGithub(unitfileParserFileLocation, podmanVersion)
	if err != nil {
		exit(fmt.Errorf("could not download unitfile.go source file: %s", err))
	}
	defer os.Remove(unitfileParserFile.Name())

	lookupFuncs, err := parseUnitFileParserSourceFile(unitfileParserFile)
	if err != nil {
		exit(fmt.Errorf("could not parse unitfile parser source file: %s", err))
	}

	quadletSourceFile, err := downloadSourceFileFromGithub(quadletFileLocation, podmanVersion)
	if err != nil {
		exit(fmt.Errorf("could not download quadlet.go source file: %s", err))
	}
	defer os.Remove(quadletSourceFile.Name())

	fieldsByGroup, err := parseQuadletSourceFile(quadletSourceFile, lookupFuncs)
	if err != nil {
		exit(fmt.Errorf("could not parse quadlet source file: %s", err))
	}

	data := sourceFileData{fieldsByGroup: fieldsByGroup, lookupFuncs: lookupFuncs}
	err = generateSourceFiles(data)
	if err != nil {
		exit(fmt.Errorf("could not generate source files: %s", err))
	}
}

func exit(err error) {
	_, err = fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	if err != nil {
		panic(err)
	}
	os.Stderr.Sync()
	os.Exit(1)
}

func getPodmanVersion(version string) string {
	if version == "" {
		if version, ok := os.LookupEnv(podmanVersionEnvKey); ok {
			return version
		}
	}

	return version
}

func downloadSourceFileFromGithub(location string, version string) (*os.File, error) {
	if version == "" {
		return nil, fmt.Errorf("podman version was not provided. "+
			"Use -%s flag or %s environment variable", podmanVersionFlag, podmanVersionEnvKey)
	}

	url := fmt.Sprintf(location, version)
	client := http.Client{}
	response, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download from '%s': %v", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status '%s' when downloading from '%s'", response.Status, url)
	}

	file, err := os.CreateTemp("", "quadlet-*.go")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file to copy the content of quadlet file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file contents: %v", err)
	}

	return file, nil
}
