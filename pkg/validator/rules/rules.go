package rules

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	. "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	model "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

const (
	ErrValueNotAllowed     = "value-not-allowed"
	ErrRequiredSuffix      = "required-suffix"
	ErrBadFormat           = "bad-format"
	ErrNoMatchRegex        = "not-match-regex"
	ErrZeroOrOneValue      = "zero-or-one-value"
	ErrOneRequired         = "one-required"
	ErrConditionNotMatched = "condition-not-matched"
)

// ================== Utilities ==================

func Rules(rules ...V.Rule) []V.Rule {
	return rules
}

func CheckRules(validator V.Validator, unit UnitFile, rules model.Groups) []V.ValidationError {
	validationErrors := make([]V.ValidationError, 0)

	groupsValue := reflect.ValueOf(rules)
	groupsType := reflect.TypeOf(rules)

	for groupIndex := range groupsType.NumField() {
		groupField := groupsType.Field(groupIndex)
		groupValue := groupsValue.Field(groupIndex)
		groupName := groupField.Name

		groupType := groupField.Type
		for fieldIndex := range groupType.NumField() {
			fieldType := groupType.Field(fieldIndex)

			fieldName := fieldType.Name

			ruleFns, _ := groupValue.FieldByName(fieldName).Interface().([]V.Rule)
			for _, rule := range ruleFns {
				field, ok := model.Fields[groupName][fieldName]
				if !ok {
					panic(fmt.Sprintf("field '%s.%s' not found in Fields map", groupName, fieldName))
				}
				field.Group = groupField.Name
				validationErrors = append(validationErrors, rule(validator, unit, field)...)
			}
		}
	}

	return validationErrors
}

// ================== Rules ==================

func RequiredIfNotPresent(other Field) V.Rule {
	return func(validator V.Validator, unit UnitFile, field Field) []V.ValidationError {
		if !unit.HasValue(other) && !unit.HasValue(field) {
			return V.RequiredKey.ErrSlice(validator.Name(), ErrOneRequired, field, 0, 0,
				fmt.Sprintf("at least one of these keys is required: %s, %s", field, other))
		}

		return nil
	}
}

func ConflictsWith(others ...Field) V.Rule {
	return func(validator V.Validator, unit UnitFile, field Field) []V.ValidationError {
		validationErrors := make([]V.ValidationError, 0)
		for _, other := range others {
			if unit.HasValue(other) && unit.HasValue(field) {
				res, _ := unit.Lookup(field)
				for _, value := range res.Values() {
					validationErrors = append(validationErrors, *V.KeyConflict.ErrForField(validator.Name(), "", field,
						value.Line, 0, fmt.Sprintf("the keys %s, %s cannot be specified together", field, other)))
				}
			}
		}

		return validationErrors
	}
}

func CanReference(unitTypes ...UnitType) V.Rule {
	return func(validator V.Validator, unit UnitFile, field Field) []V.ValidationError {
		context := validator.Context()
		if !context.CheckReferences {
			return nil
		}

		res, found := unit.Lookup(field)
		if !found || len(res.Values()) == 0 {
			return nil
		}

		units := context.AllUnitFiles
		validationErrors := make([]V.ValidationError, 0)
		for _, value := range res.Values() {
			for _, unitType := range unitTypes {
				if strings.HasSuffix(value.Value, unitType.Ext) {
					foundUnit := slices.ContainsFunc(units, func(unit UnitFile) bool {
						return unit.FileName() == value.Value
					})

					if !foundUnit {
						validationErrors = append(validationErrors, *V.InvalidReference.ErrForField(validator.Name(), "",
							field, value.Line, value.Column, fmt.Sprintf("requested Quadlet %s '%s' was not found",
								unitType.Name, value.Value)))
					}
					break
				}
			}
		}

		return validationErrors
	}
}

func HaveFormat(format Format) V.Rule {
	return func(validator V.Validator, unit UnitFile, field Field) []V.ValidationError {
		res, found := unit.Lookup(field)
		if !found {
			return nil
		}

		validationErrors := make([]V.ValidationError, 0)
		for _, value := range res.Values() {
			err := format.ParseAndValidate(value.Value)
			if err != nil {
				validationErrors = append(validationErrors, *V.InvalidValue.ErrForField(validator.Name(), ErrBadFormat, field,
					value.Line, value.Column, err.Error()))
			}
		}

		return validationErrors
	}
}

func AllowedValues(allowedValues ...string) V.Rule {
	return func(validator V.Validator, unit UnitFile, field Field) []V.ValidationError {
		res, found := unit.Lookup(field)
		if !found {
			return nil
		}

		validationErrors := make([]V.ValidationError, 0)
		for _, value := range res.Values() {
			if !slices.Contains(allowedValues, value.Value) {
				validationErrors = append(validationErrors, *V.InvalidValue.ErrForField(validator.Name(), ErrValueNotAllowed, field,
					value.Line, value.Column, fmt.Sprintf("invalid value '%s' for key '%s'. Allowed values: %s",
						value.Value, field, allowedValues)))
			}
		}
		return validationErrors
	}
}

func HasSuffix(suffix string) V.Rule {
	return func(validator V.Validator, unit UnitFile, field Field) []V.ValidationError {
		res, found := unit.Lookup(field)
		if !found {
			return nil
		}

		validationErrors := make([]V.ValidationError, 0)
		for _, value := range res.Values() {
			if !strings.HasSuffix(value.Value, suffix) {
				validationErrors = append(validationErrors, *V.InvalidValue.ErrForField(validator.Name(), ErrRequiredSuffix, field,
					value.Line, value.Column, fmt.Sprintf("value '%s' must have suffix '%s'", value.Value, suffix)))
			}
		}

		return validationErrors
	}
}

func DependsOn(dependency Field) V.Rule {
	return func(validator V.Validator, unit UnitFile, field Field) []V.ValidationError {
		dependencyRes, dependencyFound := unit.Lookup(dependency)
		dependencyOk := dependencyFound && len(dependencyRes.Values()) > 0

		res, found := unit.Lookup(field)
		fieldOk := found && len(res.Values()) > 0

		validationErrors := make([]V.ValidationError, 0)
		if !dependencyOk && fieldOk {
			for _, value := range res.Values() {
				validationErrors = append(validationErrors, *V.UnsatisfiedDependency.ErrForField(validator.Name(), "",
					field, value.Line, 0,
					fmt.Sprintf("value for '%s' was set but it depends on key '%s' which was not found",
						field, dependency.Key)))
			}
		}

		return validationErrors
	}
}

func Deprecated(validator V.Validator, unit UnitFile, field Field) []V.ValidationError {
	res, found := unit.Lookup(field)
	if !found {
		return nil
	}

	validationErrors := make([]V.ValidationError, 0)
	for _, value := range res.Values() {
		validationErrors = append(validationErrors, *V.DeprecatedKey.ErrForField(validator.Name(), "", field,
			value.Line, 0, fmt.Sprintf("key '%s' is deprecated and should not be used", field)))
	}
	return validationErrors
}

func MatchRegexp(regex *regexp.Regexp) V.Rule {
	return func(validator V.Validator, unit UnitFile, field Field) []V.ValidationError {
		res, found := unit.Lookup(field)
		if !found {
			return nil
		}

		validationErrors := make([]V.ValidationError, 0)
		for _, value := range res.Values() {
			if !regex.MatchString(value.Value) {
				validationErrors = append(validationErrors, *V.InvalidValue.ErrForField(validator.Name(), ErrNoMatchRegex, field,
					value.Line, value.Column, fmt.Sprintf("Must match regexp '%s'", regex.String())))
			}
		}
		return validationErrors
	}
}

func ValuesMust(valuesPredicate ValuesValidator, rulePredicate RulePredicate, messageAndArgs ...any) V.Rule {
	return func(validator V.Validator, unit UnitFile, field Field) []V.ValidationError {
		if !rulePredicate(validator, unit, field) {
			return nil
		}

		if res, ok := unit.Lookup(field); ok {
			if err := valuesPredicate(validator, field, res.Values()); err != nil {
				errorMsg := buildErrorMessage(messageAndArgs, err)
				var line, column int
				if len(res.Values()) > 0 {
					firstValue := res.Values()[0]
					line = firstValue.Line
					column = firstValue.Column
				}
				return V.InvalidValue.ErrSlice(validator.Name(), ErrConditionNotMatched, field, line, column, errorMsg)
			}
		}

		return nil
	}
}

func buildErrorMessage(messageAndArgs []any, err *V.ValidationError) string {
	var errorMsg string
	if len(messageAndArgs) >= 1 {
		if msg, ok := messageAndArgs[0].(string); ok {
			errorMsg = msg
			if len(messageAndArgs) > 1 {
				errorMsg = fmt.Sprintf(errorMsg, messageAndArgs[1:]...)
			}
		}
	}

	if len(errorMsg) > 0 {
		errorMsg = fmt.Sprintf("%s. %s", errorMsg, err)
	} else if len(errorMsg) == 0 {
		errorMsg = err.Error.Error()
	}

	return errorMsg
}

// ================== ValuesValidators ==================

type ValuesValidator func(validator V.Validator, field Field, values []UnitValue) *V.ValidationError
type RulePredicate func(validator V.Validator, unit UnitFile, field Field) bool

func HaveZeroOrOneValues(validator V.Validator, field Field, values []UnitValue) *V.ValidationError {
	if len(values) > 1 {
		value := values[1]
		return V.InvalidValue.ErrForField(validator.Name(), ErrZeroOrOneValue, field, value.Line, value.Column,
			"should have exactly zero or one value")
	}

	return nil
}

func WhenFieldEquals(conditionField Field, conditionValues ...string) RulePredicate {
	return func(_ V.Validator, unit UnitFile, _ Field) bool {
		if res, ok := unit.Lookup(conditionField); ok {
			for _, fieldValue := range res.Values() {
				for _, conditionValue := range conditionValues {
					if fieldValue.Value == conditionValue {
						return true
					}
				}
			}
		}

		return false
	}
}
