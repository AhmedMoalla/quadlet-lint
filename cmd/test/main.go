package main

import (
	"flag"
	"fmt"
	"github.com/containers/podman/v5/pkg/systemd/parser"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

func readInputPath() string {
	flag.Parse()

	var inputDirOrFile string
	if flag.NArg() == 0 {
		inputDirOrFile = getWorkingDirectory()
	} else {
		inputDirOrFile = flag.Arg(0)
	}
	return inputDirOrFile
}

func findUnitFiles(inputDirOrFile string) []string {
	var unitFilesPaths []string
	if isDir(inputDirOrFile) {
		unitFilesPaths = getAllUnitFiles(inputDirOrFile)
	} else {
		unitFilesPaths = []string{inputDirOrFile}
	}
	return unitFilesPaths
}

func parseUnitFiles(unitFilesPaths []string) []parser.UnitFile {
	unitFiles := make([]parser.UnitFile, len(unitFilesPaths))
	for _, path := range unitFilesPaths {
		unitFile, err := parser.ParseUnitFile(path)
		if err != nil {
			reportError(path, err)
		}
		unitFiles = append(unitFiles, *unitFile)
	}
	return unitFiles
}

func main() {
	inputPath := readInputPath()
	unitFilesPaths := findUnitFiles(inputPath)
	if len(unitFilesPaths) == 0 {
		fmt.Printf("no unit files were found in %s\n", inputPath)
		os.Exit(0)
	}

	unitFiles := parseUnitFiles(unitFilesPaths)

	for _, file := range unitFiles {
		fmt.Println(file.Filename)
	}

	reportErrors()
}

func getWorkingDirectory() string {
	executable, err := os.Executable()
	if err != nil {
		panic(err)
	}

	return filepath.Dir(executable)
}

func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		panic(err)
	}

	return fileInfo.IsDir()
}

var supportedExtensions = []string{
	".container", ".volume", ".kube", ".network", ".image", ".build", ".pod",
}

func getAllUnitFiles(rootDirectory string) []string {
	unitFilesPaths := make([]string, 0)
	err := filepath.WalkDir(rootDirectory, func(path string, entry fs.DirEntry, err error) error {
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}

		if slices.Contains(supportedExtensions, filepath.Ext(path)) {
			unitFilesPaths = append(unitFilesPaths, path)
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	return unitFilesPaths
}

var errors = make(map[string][]error)

func reportError(path string, err error) {
	if _, present := errors[path]; !present {
		errors[path] = make([]error, 0, 1)
	}
	errors[path] = append(errors[path], err)
}

func reportErrors() {
	if len(errors) > 0 {
		fmt.Println("Following errors have been found")
		for path, errs := range errors {
			fmt.Printf("%s:\n", path)
			for _, err := range errs {
				fmt.Printf("\t-> %s\n", err)
			}
		}
	}
}
