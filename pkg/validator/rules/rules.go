package rules

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	. "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	model "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	P "github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

// ================== Utilities ==================

func Rules(rules ...V.Rule) []V.Rule {
	return rules
}

func ErrSlice(validatorName string, errType V.ErrorType, line, column int, message string) []V.ValidationError {
	return []V.ValidationError{*V.Err(validatorName, errType, line, column, message)}
}

func CheckRules(validator V.Validator, unit P.UnitFile, rules model.Groups) []V.ValidationError {
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
					panic(fmt.Sprintf("field %s not found in Fields map", fieldName))
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
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		if !unit.HasValue(other) && !unit.HasValue(field) {
			return ErrSlice(validator.Name(), V.RequiredKey, 0, 0,
				fmt.Sprintf("at least one of these keys is required: %s, %s", field, other))
		}

		return nil
	}
}

func ConflictsWith(others ...Field) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		validationErrors := make([]V.ValidationError, 0)
		for _, other := range others {
			if unit.HasValue(other) && unit.HasValue(field) {
				res, _ := unit.Lookup(field)
				for _, value := range res.Values {
					validationErrors = append(validationErrors, *V.Err(validator.Name(), V.KeyConflict, value.Line, 0,
						fmt.Sprintf("the keys %s, %s cannot be specified together", field, other)))
				}
			}
		}

		return validationErrors
	}
}

func CanReference(unitTypes ...P.UnitType) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		context := validator.Context()
		if !context.CheckReferences {
			return nil
		}

		res, found := unit.Lookup(field)
		if !found || len(res.Values) == 0 {
			return nil
		}

		units := context.AllUnitFiles
		validationErrors := make([]V.ValidationError, 0)
		for _, value := range res.Values {
			for _, unitType := range unitTypes {
				if strings.HasSuffix(value.Value, unitType.Ext) {
					foundUnit := slices.ContainsFunc(units, func(unit P.UnitFile) bool {
						return unit.Filename == value.Value
					})

					if !foundUnit {
						validationErrors = append(validationErrors, *V.Err(validator.Name(), V.InvalidReference, value.Line, value.Column,
							fmt.Sprintf("requested Quadlet %s '%s' was not found", unitType.Name, value.Value)))
					}
					break
				}
			}
		}

		return validationErrors
	}
}

func HaveFormat(format Format) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		res, found := unit.Lookup(field)
		if !found {
			return nil
		}

		validationErrors := make([]V.ValidationError, 0)
		for _, value := range res.Values {
			err := format.ParseAndValidate(value.Value)
			if err != nil {
				validationErrors = append(validationErrors, *V.Err(validator.Name(), V.InvalidValue, value.Line,
					value.Column, err.Error()))
			}
		}

		return validationErrors
	}
}

func AllowedValues(allowedValues ...string) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		res, found := unit.Lookup(field)
		if !found {
			return nil
		}

		validationErrors := make([]V.ValidationError, 0)
		for _, value := range res.Values {
			if !slices.Contains(allowedValues, value.Value) {
				validationErrors = append(validationErrors, *V.Err(validator.Name(), V.InvalidValue, value.Line, value.Column,
					fmt.Sprintf("invalid value '%s' for key '%s'. Allowed values: %s",
						value.Value, field, allowedValues)))
			}
		}
		return validationErrors
	}
}

func HasSuffix(suffix string) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		res, found := unit.Lookup(field)
		if !found {
			return nil
		}

		validationErrors := make([]V.ValidationError, 0)
		for _, value := range res.Values {
			if !strings.HasSuffix(value.Value, suffix) {
				validationErrors = append(validationErrors, *V.Err(validator.Name(), V.InvalidValue, value.Line, value.Column,
					fmt.Sprintf("value '%s' must have suffix '%s'", value.Value, suffix)))
			}
		}

		return validationErrors
	}
}

func DependsOn(dependency Field) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		dependencyRes, dependencyFound := unit.Lookup(dependency)
		dependencyOk := dependencyFound && len(dependencyRes.Values) > 0

		res, found := unit.Lookup(field)
		fieldOk := found && len(res.Values) > 0

		validationErrors := make([]V.ValidationError, 0)
		if !dependencyOk && fieldOk {
			for _, value := range res.Values {
				validationErrors = append(validationErrors, *V.Err(validator.Name(), V.UnsatisfiedDependency, value.Line, 0,
					fmt.Sprintf("value for '%s' was set but it depends on key '%s' which was not found",
						field, dependency.Key)))
			}
		}

		return validationErrors
	}
}

func Deprecated(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
	res, found := unit.Lookup(field)
	if !found {
		return nil
	}

	validationErrors := make([]V.ValidationError, 0)
	for _, value := range res.Values {
		validationErrors = append(validationErrors, *V.Err(validator.Name(), V.DeprecatedKey, value.Line, 0,
			fmt.Sprintf("key '%s' is deprecated and should not be used", field)))
	}
	return validationErrors
}

func MatchRegexp(regex *regexp.Regexp) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		res, found := unit.Lookup(field)
		if !found {
			return nil
		}

		validationErrors := make([]V.ValidationError, 0)
		for _, value := range res.Values {
			if !regex.MatchString(value.Value) {
				validationErrors = append(validationErrors, *V.Err(validator.Name(), V.InvalidValue, value.Line, value.Column,
					fmt.Sprintf("Must match regexp '%s'", regex.String())))
			}
		}
		return validationErrors
	}
}

func ValuesMust(valuesPredicate ValuesValidator, rulePredicate RulePredicate, messageAndArgs ...any) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		if !rulePredicate(validator, unit, field) {
			return nil
		}

		if res, ok := unit.Lookup(field); ok {
			if err := valuesPredicate(validator, field, res.Values); err != nil {
				errorMsg := buildErrorMessage(messageAndArgs, err)
				var line, column int
				if len(res.Values) > 0 {
					firstValue := res.Values[0]
					line = firstValue.Line
					column = firstValue.Column
				}
				return ErrSlice(validator.Name(), V.InvalidValue, line, column, errorMsg)
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
		errorMsg = err.Error()
	}

	return errorMsg
}

// ================== ValuesValidators ==================

type ValuesValidator func(validator V.Validator, field Field, values []P.UnitValue) *V.ValidationError
type RulePredicate func(validator V.Validator, unit P.UnitFile, field Field) bool

func HaveZeroOrOneValues(validator V.Validator, _ Field, values []P.UnitValue) *V.ValidationError {
	if len(values) > 1 {
		value := values[1]
		return V.Err(validator.Name(), V.InvalidValue, value.Line, value.Column, "should have exactly zero or one value")
	}

	return nil
}

func WhenFieldEquals(conditionField Field, conditionValues ...string) RulePredicate {
	return func(_ V.Validator, unit P.UnitFile, _ Field) bool {
		if res, ok := unit.Lookup(conditionField); ok {
			for _, fieldValue := range res.Values {
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
