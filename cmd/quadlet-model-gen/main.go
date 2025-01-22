package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	podmanVersionFlag    = "podman-version"
	podmanVersionEnvKey  = "PODMAN_VERSION"
	fallbackFilesDirFlag = "fallback-dir"

	baseModelPackageName = "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
)

var (
	podmanVersion = flag.String(podmanVersionFlag, "", "Podman's tag used to download the source file for code generation")
	fallbackDir   = flag.String(fallbackFilesDirFlag, "", "Fallback files directory")
)

func main() {
	outputDir, err := getOutputDir()
	if err != nil {
		exit(err)
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		exit(err)
	}

	if len(entries) != 0 {
		return
	}

	flag.Parse()

	podmanVersion := getPodmanVersion(*podmanVersion)
	if podmanVersion == "" {
		exit(fmt.Errorf("podman version was not provided. "+
			"Use -%s flag or %s environment variable", podmanVersionFlag, podmanVersionEnvKey))
	}

	var fallbackMap map[DownloadableFile]string
	if *fallbackDir != "" {
		absFallbackDir, err := filepath.Abs(*fallbackDir)
		if err != nil {
			exit(err)
		}

		fallbackMap = map[DownloadableFile]string{
			Quadlet:  filepath.Join(absFallbackDir, podmanVersion, "quadlet.go"),
			Unitfile: filepath.Join(absFallbackDir, podmanVersion, "unitfile.go"),
		}
	}

	fmt.Printf("============== Quadlet Model Generation =============\n")
	fmt.Printf("Output Dir: %s\n", outputDir)
	fmt.Printf("Podman version: %s\n", podmanVersion)
	fmt.Printf("Quadlet Fallback: %v\n", fallbackMap[Quadlet])
	fmt.Printf("Unitfile Fallback: %v\n", fallbackMap[Unitfile])
	fmt.Printf("=====================================================\n")

	downloader := NewDownloader(podmanVersion, fallbackMap)

	unitfileParserFile, err := downloader.Download(Unitfile)
	if err != nil {
		exit(fmt.Errorf("could not download unitfile.go source file: %w", err))
	}
	defer unitfileParserFile.Close()
	defer os.Remove(unitfileParserFile.Name())

	quadletSourceFile, err := downloader.Download(Quadlet)
	if err != nil {
		exit(fmt.Errorf("could not download quadlet.go source file: %w", err))
	}
	defer quadletSourceFile.Close()
	defer os.Remove(quadletSourceFile.Name())

	parseAndGenerateFiles(quadletSourceFile, unitfileParserFile)
}

func parseAndGenerateFiles(quadletSourceFile, unitfileParserFile *os.File) {
	lookupFuncs, err := parseUnitFileParserSourceFile(unitfileParserFile)
	if err != nil {
		exit(fmt.Errorf("could not parse unitfile parser source file: %w", err))
	}

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
