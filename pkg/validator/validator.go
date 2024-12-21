package validator

import (
	"fmt"

	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
)

type Validator interface {
	Name() string
	Context() Context
	Validate(unit parser.UnitFile) []ValidationError
}

type Context struct {
	Options
	AllUnitFiles []parser.UnitFile
}

type Options struct {
	CheckReferences bool
}

var (
	UnknownKey            = NewErrorType("unknown-key", LevelError)
	RequiredKey           = NewErrorType("required-key", LevelError)
	KeyConflict           = NewErrorType("key-conflict", LevelError)
	InvalidValue          = NewErrorType("invalid-value", LevelError)
	DeprecatedKey         = NewErrorType("deprecated-key", LevelWarning)
	UnsatisfiedDependency = NewErrorType("unsatisfied-dependency", LevelError)
)

type ValidationError struct {
	ErrorType
	Location
	Message       string
	ValidatorName string
}

func Err(validatorName string, errType ErrorType, line, column int, message string) *ValidationError {
	return &ValidationError{
		ErrorType:     errType,
		Location:      Location{Line: line, Column: column},
		Message:       message,
		ValidatorName: validatorName,
	}
}

func (err ValidationError) Error() string {
	return err.Message
}

func (err ValidationError) String() string {
	return fmt.Sprintf("%s.%s", err.ValidatorName, err.ErrorType.Name)
}

type ErrorType struct {
	Name          string
	Level         Level
	ValidatorName string
}

func NewErrorType(name string, level Level) ErrorType {
	return ErrorType{
		Name:  name,
		Level: level,
	}
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
	return len(errors.WhereLevel(LevelError)) > 0 || len(errors.WhereLevel(LevelWarning)) > 0
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
