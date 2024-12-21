package main

import (
	"flag"
	"fmt"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator/quadlet"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

// args
var (
	debug           bool
	checkReferences bool // true if -check-references is passed
)

func init() {
	flag.BoolVar(&debug, "debug", false, "Print debug logs")
	flag.BoolVar(&checkReferences, "check-references", false, "Check references to other Quadlet files")
}

func main() {
	flag.Parse()

	inputPath := readInputPath()
	unitFilesPaths := findUnitFiles(inputPath)
	if len(unitFilesPaths) == 0 {
		fmt.Printf("no unit files were found in %s\n", inputPath)
		os.Exit(0)
	}

	unitFiles, parsingErrors := parseUnitFiles(unitFilesPaths)

	validationErrors := validateUnitFiles(unitFiles, checkReferences)

	errors := validationErrors.Merge(parsingErrors)
	reportErrors(errors)

	logSummary(unitFilesPaths, errors)
}

func logSummary(unitFiles []string, errors validator.ValidationErrors) {
	var status string
	if errors.HasErrors() {
		status = "Failed"
	} else {
		status = "Passed"
	}

	fmt.Printf("%s: %d error(s), %d warning(s) on %d files.\n", status,
		len(errors.WhereLevel(validator.LevelError)), len(errors.WhereLevel(validator.LevelWarning)), len(unitFiles))
}

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

var ParsingError = validator.ErrorType{
	Name:          "parsing-error",
	Level:         validator.LevelError,
	ValidatorName: "parser",
}

func parseUnitFiles(unitFilesPaths []string) ([]parser.UnitFile, validator.ValidationErrors) {
	errors := make(validator.ValidationErrors)
	unitFiles := make([]parser.UnitFile, 0, len(unitFilesPaths))
	for _, path := range unitFilesPaths {
		unitFile, errs := parser.ParseUnitFile(path)
		if unitFile != nil {
			unitFiles = append(unitFiles, *unitFile)
		}

		for _, err := range errs {
			errors.AddError(path, *validator.Err("", ParsingError, err.Line, err.Column, err.Error()))
		}
	}
	return unitFiles, errors
}

func validateUnitFiles(unitFiles []parser.UnitFile, checkReferences bool) validator.ValidationErrors {
	validationErrors := make(validator.ValidationErrors)
	validators := []validator.Validator{
		quadlet.Validator(unitFiles, validator.Options{CheckReferences: checkReferences}),
	}

	for _, file := range unitFiles {
		for _, vtor := range validators {
			validationErrors.AddError(file.Filename, vtor.Validate(file)...)
		}
	}
	return validationErrors
}

func reportErrors(errors validator.ValidationErrors) {
	if errors.HasErrors() {
		fmt.Println("Following errors have been found")
		for path, errs := range errors {
			if len(errs) == 0 {
				continue
			}

			fmt.Printf("%s:\n", path)
			for _, err := range errs {
				fmt.Printf("\t-> [%s][%s.%s][%d:%d] %s\n", err.Level, err.ValidatorName, err.ErrorType.Name, err.Line, err.Column,
					err.Message)
			}
		}
	}
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
