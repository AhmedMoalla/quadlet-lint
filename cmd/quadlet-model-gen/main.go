package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	podmanVersionFlag    = "podman-version"
	podmanVersionEnvKey  = "PODMAN_VERSION"
	quadletFileLocation  = "https://raw.githubusercontent.com/containers/podman/refs/tags/%s/pkg/systemd/quadlet/quadlet.go"
	baseModelPackageName = "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
)

var (
	podmanVersion = flag.String(podmanVersionFlag, "", "Podman's tag used to download the source file for code generation")
)

// TODO: gofmt generated files
// Download the quadlet.go file from given podman tag version
//
//	Use: https://raw.githubusercontent.com/containers/podman/refs/tags/v5.3.1/pkg/systemd/quadlet/quadlet.go
//
// Parse the group variables to extract group names
// Create a groups.go file and create the Groups struct in it
// For each group:
// - Generate a package with a single file under the validator/generated package
// - In the file:
//   - Create a struct with the group's name and add a field of type []V.Rule for each supported key (Supported keys are in the maps, e.g.: supportedContainerKeys for Container group for example)
//   - Create a variable of type P.Field for each supported field encountered
//
// - In the groups.go file:
//   - Add a field for the group pointing to the structure created previously
//   - Add all the fields in the Fields map
func main() {
	flag.Parse()

	podmanVersion := getPodmanVersion(*podmanVersion)
	quadletSourceFile, err := downloadQuadletSourceFileFromGithub(podmanVersion)
	if err != nil {
		exit(fmt.Errorf("could not download quadlet quadletSourceFile: %s", err))
	}
	defer os.Remove(quadletSourceFile.Name())

	data, err := parseQuadletSourceFile(quadletSourceFile)
	if err != nil {
		exit(fmt.Errorf("could not parse quadlet source file: %s", err))
	}

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

func downloadQuadletSourceFileFromGithub(version string) (*os.File, error) {
	if version == "" {
		return nil, fmt.Errorf("podman version was not provided. "+
			"Use -%s flag or %s environment variable", podmanVersionFlag, podmanVersionEnvKey)
	}

	url := fmt.Sprintf(quadletFileLocation, version)
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
