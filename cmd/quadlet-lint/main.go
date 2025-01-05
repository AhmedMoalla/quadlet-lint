package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator/common"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator/quadlet"
)

var (
	debug           = flag.Bool("debug", false, "Enable debug logging")
	checkReferences = flag.Bool("check-references", false, "Enable checking references to other Quadlet files")
)

func main() {
	flag.Parse()

	initializeLogging(*debug)

	inputPath := readInputPath()
	unitFilesPaths := findUnitFiles(inputPath)
	if len(unitFilesPaths) == 0 {
		fmt.Fprintf(os.Stderr, "no unit files were found in %s\n", inputPath)
		os.Exit(0)
	}

	unitFiles, parsingErrors := parseUnitFiles(unitFilesPaths)

	validationErrors := validateUnitFiles(unitFiles, *checkReferences)

	errors := validationErrors.Merge(parsingErrors)
	reportErrors(errors)

	logSummary(unitFilesPaths, errors)
}

func initializeLogging(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == "time" || attr.Key == "level" {
				return slog.Attr{}
			}

			return attr
		},
	}))

	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.SetDefault(logger)

	if debug {
		slog.Debug("debug logging enabled")
		flags := make(map[string]flag.Value, flag.NFlag())
		flag.VisitAll(func(f *flag.Flag) {
			flags[f.Name] = f.Value
		})
		slog.Debug("flags have been passed to quadlet-lint", "flags", flags)
	}
}

func logSummary(unitFiles []string, errors validator.ValidationErrors) {
	var status string
	if errors.HasErrors() {
		status = "Failed"
	} else {
		status = "Passed"
	}

	fmt.Fprintf(os.Stderr, "%s: %d error(s), %d warning(s) on %d files.\n", status,
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
	slog.Debug("read input directory", "inputDir", inputDirOrFile)
	return inputDirOrFile
}

func findUnitFiles(inputDirOrFile string) []string {
	var unitFilesPaths []string
	if isDir(inputDirOrFile) {
		unitFilesPaths = getAllUnitFiles(inputDirOrFile)
	} else {
		unitFilesPaths = []string{inputDirOrFile}
	}
	slog.Debug("found unit files", "unitFiles", unitFilesPaths)
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
			errors.AddError(path, *validator.Err("parser", ParsingError, err.Line, err.Column, err.Error()))
		}
	}

	log.Printf("parsed %d unit files", len(unitFiles))
	return unitFiles, errors
}

func validateUnitFiles(unitFiles []parser.UnitFile, checkReferences bool) validator.ValidationErrors {
	validationErrors := make(validator.ValidationErrors)
	validators := []validator.Validator{
		common.Validator(),
		quadlet.Validator(unitFiles, validator.Options{CheckReferences: checkReferences}),
	}

	if *debug {
		names := make([]string, 0, len(validators))
		for _, v := range validators {
			names = append(names, v.Name())
		}
		slog.Debug("validating unit files", "validators", names)
	}

	for _, file := range unitFiles {
		for _, vtor := range validators {
			validationErrors.AddError(file.FilePath, vtor.Validate(file)...)
		}
	}
	return validationErrors
}

func reportErrors(errors validator.ValidationErrors) {
	if errors.HasErrors() {
		fmt.Fprintf(os.Stderr, "Following errors have been found:\n")
		for path, errs := range errors {
			if len(errs) == 0 {
				continue
			}

			for _, err := range errs {
				fmt.Fprintf(os.Stderr, "%s:%d:%d:%s [%s.%s] - %s\n", path, err.Line, err.Column, err.Level, err.ValidatorName,
					err.ErrorType.Name, err.Message)
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
	parser.UnitTypeContainer.Ext,
	parser.UnitTypeVolume.Ext,
	parser.UnitTypeKube.Ext,
	parser.UnitTypeNetwork.Ext,
	parser.UnitTypeImage.Ext,
	parser.UnitTypeBuild.Ext,
	parser.UnitTypePod.Ext,
}

func getAllUnitFiles(rootDirectory string) []string {
	unitFilesPaths := make([]string, 0)
	err := filepath.WalkDir(rootDirectory, func(path string, entry fs.DirEntry, err error) error {
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		if slices.Contains(supportedExtensions, filepath.Ext(path)) {
			unitFilesPaths = append(unitFilesPaths, path)
		}

		slog.Debug("file was skipped while looking for unit files", "path", path)

		return nil
	})

	if err != nil {
		panic(err)
	}

	return unitFilesPaths
}
