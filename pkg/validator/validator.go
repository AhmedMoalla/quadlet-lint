package validator

import "github.com/AhmedMoalla/quadlet-lint/pkg/parser"

type Validator interface {
	Validate(unitFile parser.UnitFile) []ValidationError
}

func Error(errType ErrorType, line, column int, message string) ValidationError {
	return ValidationError{
		ErrorType: errType,
		Location:  Location{Line: line, Column: column},
		Message:   message,
	}
}

type ValidationError struct {
	ErrorType
	Location
	Message string
}

type ErrorType struct {
	Name          string
	Level         Level
	ValidatorName string
}

type Location struct {
	FilePath string
	Line     int
	Column   int
}

type Level string

const (
	LevelError   Level = "error"
	LevelWarning Level = "warning"
)

type ValidationErrors map[string][]ValidationError

func (errors ValidationErrors) WhereLevel(level Level) []ValidationError {
	levelErrors := make([]ValidationError, 0)

	for _, errs := range errors {
		for _, err := range errs {
			if err.Level == level {
				levelErrors = append(levelErrors, err)
			}
		}
	}

	return levelErrors
}

func (errors ValidationErrors) HasErrors() bool {
	return len(errors.WhereLevel(LevelError)) > 0
}

func (errors ValidationErrors) AddError(filePath string, err ...ValidationError) {
	if _, present := errors[filePath]; !present {
		errors[filePath] = make([]ValidationError, 0, len(err))
	}
	errors[filePath] = append(errors[filePath], err...)
}

func (errors ValidationErrors) Merge(other ValidationErrors) ValidationErrors {
	for filePath, err := range other {
		errors.AddError(filePath, err...)
	}
	return errors
}
