package validator

import "github.com/AhmedMoalla/quadlet-lint/pkg/parser"

type Validator interface {
	Validate(unitFile parser.UnitFile) []ValidationError
}

type ValidationError struct {
	FilePath      string
	Position      Position
	Level         Level
	ErrorType     ErrorType
	ValidatorName string
	Message       string
}

type Position struct {
	Line   int
	Column int
}

type Level string

const (
	Error   Level = "error"
	Warning Level = "warning"
)

type ErrorType string

const (
	ParsingError ErrorType = "parsing-error"
)

type ValidationErrors map[string][]ValidationError

func (errors ValidationErrors) Level(level Level) []ValidationError {
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
