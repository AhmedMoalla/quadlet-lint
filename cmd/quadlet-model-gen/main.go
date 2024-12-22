package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

const quadletFileLocation = "https://raw.githubusercontent.com/containers/podman/refs/tags/%s/pkg/systemd/quadlet/quadlet.go"

var (
	podmanVersion = flag.String("podman-version", "", "Podman's tag used to download the source file for code generation")
)

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

	file, err := downloadPodmanQuadletSourceFile(podmanVersion)
	if err != nil {
		panic(fmt.Sprintf("could not download quadlet file: %s", err))
	}
	defer os.Remove(file.Name())

}

func downloadPodmanQuadletSourceFile(version *string) (*os.File, error) {
	var tag string
	if version == nil {
		if version, ok := os.LookupEnv("PODMAN_VERSION"); ok {
			tag = version
		} else {
			return nil, errors.New("podman version was not provided. Use -podman-version")
		}
	} else {
		tag = *version
	}

	url := fmt.Sprintf(quadletFileLocation, tag)
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
