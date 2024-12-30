package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	podmanVersionFlag   = "podman-version"
	podmanVersionEnvKey = "PODMAN_VERSION"

	podmanGithubTagsURL        = "https://raw.githubusercontent.com/containers/podman/refs/tags/%s"
	quadletFileLocation        = podmanGithubTagsURL + "/pkg/systemd/quadlet/quadlet.go"
	unitfileParserFileLocation = podmanGithubTagsURL + "/pkg/systemd/parser/unitfile.go"
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
		exit(fmt.Errorf("could not download unitfile.go source file: %w", err))
	}
	defer os.Remove(unitfileParserFile.Name())

	lookupFuncs, err := parseUnitFileParserSourceFile(unitfileParserFile)
	if err != nil {
		exit(fmt.Errorf("could not parse unitfile parser source file: %w", err))
	}

	quadletSourceFile, err := downloadSourceFileFromGithub(quadletFileLocation, podmanVersion)
	if err != nil {
		exit(fmt.Errorf("could not download quadlet.go source file: %w", err))
	}
	defer os.Remove(quadletSourceFile.Name())

	fieldsByGroup, err := parseQuadletSourceFile(quadletSourceFile, lookupFuncs)
	if err != nil {
		exit(fmt.Errorf("could not parse quadlet source file: %w", err))
	}

	data := sourceFileData{fieldsByGroup: fieldsByGroup, lookupFuncs: lookupFuncs}
	err = generateSourceFiles(data)
	if err != nil {
		exit(fmt.Errorf("could not generate source files: %w", err))
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
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download from '%s': %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status '%s' when downloading from '%s'", response.Status, url)
	}

	file, err := os.CreateTemp("", "quadlet-*.go")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file to copy the content of quadlet file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file contents: %w", err)
	}

	return file, nil
}
