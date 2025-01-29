package validator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model"
)

type Rule = func(validator Validator, unit model.UnitFile, field model.Field) []ValidationError

type Validator interface {
	Name() string
	Context() Context
	Validate(unit model.UnitFile) []ValidationError
}

type Context struct {
	Options
	AllFields    model.FieldsMap
	AllUnitFiles []model.UnitFile
}

type Options struct {
	CheckReferences bool
}

var (
	UnknownKey            = NewErrorCategory("unknown-key", LevelError)
	RequiredKey           = NewErrorCategory("required-key", LevelError)
	KeyConflict           = NewErrorCategory("key-conflict", LevelError)
	InvalidValue          = NewErrorCategory("invalid-value", LevelError)
	DeprecatedKey         = NewErrorCategory("deprecated-key", LevelWarning)
	UnsatisfiedDependency = NewErrorCategory("unsatisfied-dependency", LevelError)
	InvalidReference      = NewErrorCategory("invalid-reference", LevelError)
)

type ValidationError struct {
	Category      ErrorCategory
	Location      Location
	Error         error
	ValidatorName string
	Group         string
	Key           string
	Name          string
}

func (err ValidationError) String() string {
	if err.Name != "" {
		return fmt.Sprintf("%s.%s.%s", err.ValidatorName, err.Category.Name, err.Name)
	}
	return fmt.Sprintf("%s.%s", err.ValidatorName, err.Category.Name)
}

type ErrorCategory struct {
	Name  string
	Level Level
}

func (c ErrorCategory) ErrForField(validatorName, errName string, field model.Field,
	line, column int, message string) *ValidationError {
	return c.ErrWithName(validatorName, errName, field.Group, field.Key, line, column, message)
}

func (c ErrorCategory) Err(validatorName string, group, key string, line, column int, message string) *ValidationError {
	return c.ErrWithName(validatorName, "", group, key, line, column, message)
}

func (c ErrorCategory) ErrWithName(validatorName, errName, group, key string,
	line, column int, message string) *ValidationError {
	var err error
	if errName == "" {
		err = fmt.Errorf("%s: %s", c.Name, message)
	} else {
		err = fmt.Errorf("%s.%s: %s", c.Name, errName, message)
	}

	return &ValidationError{
		Category:      c,
		Location:      Location{Line: line, Column: column},
		Error:         err,
		ValidatorName: validatorName,
		Group:         group,
		Key:           key,
		Name:          errName,
	}
}

func (c ErrorCategory) ErrSlice(validatorName, errName string,
	field model.Field, line, column int, message string) []ValidationError {
	return []ValidationError{*c.ErrForField(validatorName, errName, field, line, column, message)}
}

func NewErrorCategory(name string, level Level) ErrorCategory {
	return ErrorCategory{
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
			if err.Category.Level == level {
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

func (errors ValidationErrors) AddEnabledError(path string, disabledErrs model.DisabledErrors, err ...ValidationError) {
	filtered := filterDisabled(disabledErrs, err...)
	errors.AddError(path, filtered...)
}

func filterDisabled(disabledErrs model.DisabledErrors, errs ...ValidationError) []ValidationError {
	filtered := make([]ValidationError, 0)
	if disabledErrs.DisableAll {
		return filtered
	}

	for _, err := range errs {
		if containsError(disabledErrs.Global, err) {
			continue
		}

		filtered = append(filtered, err)
	}

	return filtered
}

func containsError(errors []string, err ValidationError) bool {
	return slices.ContainsFunc(errors, func(error string) bool {
		parts := strings.Split(error, ".")
		switch len(parts) {
		case 1:
			return err.Name == error
		case 2:
			return err.Category.Name == parts[0] && err.Name == parts[1]
		case 3:
			return err.ValidatorName == parts[0] && err.Category.Name == parts[1] && err.Name == parts[2]
		default:
			return false // Log a warning because of malformed error ?
		}
	})
}

func (errors ValidationErrors) Merge(other ValidationErrors) ValidationErrors {
	for filePath, err := range other {
		errors.AddError(filePath, err...)
	}
	return errors
}
