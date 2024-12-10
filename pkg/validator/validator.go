package validator

import "github.com/containers/podman/v5/pkg/systemd/parser"

type ValidationErrorLevel string

const (
	Error   ValidationErrorLevel = "error"
	Warning ValidationErrorLevel = "warning"
)

type ValidationErrorType string

const (
	ParsingError ValidationErrorType = "parsing-error"
)

type ValidationError struct {
	FilePath      string
	Position      ValidationErrorPosition
	Level         ValidationErrorLevel
	ErrorType     ValidationErrorType
	ValidatorName string
	Message       string
}

type ValidationErrorPosition struct {
	Line   int
	Column int
}

type ValidationErrors map[string][]ValidationError

func (errors ValidationErrors) Level(level ValidationErrorLevel) []ValidationError {
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
	for _, errs := range errors {
		if len(errs) > 0 {
			return true
		}
	}
	return false
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

type Validator interface {
	Validate(unitFile parser.UnitFile) []ValidationError
}
