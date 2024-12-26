package rules

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	. "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
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

// TODO: Refactor
func CheckRules(validator V.Validator, unit P.UnitFile, rules Groups) []V.ValidationError {
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

			ruleFns := groupValue.FieldByName(fieldName).Interface().([]V.Rule)
			for _, rule := range ruleFns {
				field, ok := Fields[groupName][fieldName]
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
		if !unit.HasValue(other.Group, other.Key) && !unit.HasValue(field.Group, field.Key) {
			value, _ := unit.Lookup(field.Group, field.Key)
			return ErrSlice(validator.Name(), V.RequiredKey, value.Line, 0,
				fmt.Sprintf("at least one of these keys is required: %s, %s", field, other))
		}

		return nil
	}
}

func ConflictsWith(others ...Field) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		validationErrors := make([]V.ValidationError, 0)
		for _, other := range others {
			if unit.HasValue(other.Group, other.Key) && unit.HasValue(field.Group, field.Key) {
				value, _ := unit.Lookup(field.Group, field.Key)
				validationErrors = append(validationErrors, *V.Err(validator.Name(), V.KeyConflict, value.Line, 0,
					fmt.Sprintf("the keys %s, %s cannot be specified together", field, other)))
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

		var values []P.UnitValue
		if field.Multiple() {
			values = unit.LookupAll(field.Group, field.Key)
		} else if value, found := unit.Lookup(field.Group, field.Key); found {
			values = []P.UnitValue{value}
		}

		if len(values) == 0 {
			return nil
		}

		units := context.AllUnitFiles
		validationErrors := make([]V.ValidationError, 0)
		for _, value := range values {
			for _, unitType := range unitTypes {
				if strings.HasSuffix(value.Value, unitType.Ext) {
					foundUnit := slices.ContainsFunc(units, func(unit P.UnitFile) bool {
						return unit.Filename == value.Value
					})

					if !foundUnit {
						validationErrors = append(validationErrors, *V.Err(validator.Name(), V.InvalidReference, value.Line, value.Column,
							fmt.Sprintf("requested Quadlet %s '%s' was not found", unitType, value)))
					}
				}

				break
			}
		}

		return validationErrors
	}
}

func HaveFormat(format Format) ValuesValidator {
	return func(validator V.Validator, field Field, values []P.UnitValue) *V.ValidationError {
		for _, value := range values {
			err := format.ParseAndValidate(value.Value)
			if err != nil {
				return V.Err(validator.Name(), V.InvalidValue, 0, 0, err.Error())
			}
		}

		return nil
	}
}

func AllowedValues(allowedValues ...string) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		value, ok := unit.Lookup(field.Group, field.Key)
		if ok && !slices.Contains(allowedValues, value.Value) {
			return ErrSlice(validator.Name(), V.InvalidValue, value.Line, value.Column,
				fmt.Sprintf("invalid value '%s' for key '%s'. Allowed values: %s",
					value, field, allowedValues))
		}
		return nil
	}
}

func HasSuffix(suffix string) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		value, found := unit.Lookup(field.Group, field.Key)
		if !found {
			return nil
		}

		if !strings.HasSuffix(value.Value, suffix) {
			return ErrSlice(validator.Name(), V.InvalidValue, value.Line, value.Column,
				fmt.Sprintf("value '%s' must have suffix '%s'", value, suffix))
		}

		return nil
	}
}

func DependsOn(dependency Field) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		dependency, dependencyFound := unit.Lookup(dependency.Group, dependency.Key)
		dependencyOk := dependencyFound && len(dependency.Value) > 0

		value, found := unit.Lookup(field.Group, field.Key)
		fieldOk := found && len(value.Value) > 0

		if !dependencyOk && fieldOk {
			return ErrSlice(validator.Name(), V.UnsatisfiedDependency, value.Line, 0,
				fmt.Sprintf("value for '%s' was set but it depends on key '%s' which was not found",
					field, dependency))
		}

		return nil
	}
}

func Deprecated(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
	if value, found := unit.Lookup(field.Group, field.Key); found {
		return ErrSlice(validator.Name(), V.DeprecatedKey, value.Line, 0,
			fmt.Sprintf("key '%s' is deprecated and should not be used", field))
	}

	return nil
}

// TODO: line and column number not implemented
func ValuesMust(valuePredicate ValuesValidator, rulePredicate RulePredicate, messageAndArgs ...any) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field Field) []V.ValidationError {
		if rulePredicate(validator, unit, field) {
			// TODO: Should use correct Lookup function depending on the field
			// Refactor Lookup function to take Field instances
			// Fields should define LookupFunc property that tells which Lookup function to use
			values := unit.LookupAllStrv(field.Group, field.Key)
			if err := valuePredicate(validator, field, values); err != nil {
				errorMsg := buildErrorMessage(messageAndArgs, err)
				return ErrSlice(validator.Name(), V.InvalidValue, 0, 0, errorMsg)
			}
		}
		return nil
	}
}

func buildErrorMessage(messageAndArgs []any, err *V.ValidationError) string {
	var errorMsg string
	if len(messageAndArgs) == 1 {
		errorMsg = messageAndArgs[0].(string)
	} else if len(messageAndArgs) > 1 {
		errorMsg = fmt.Sprintf(messageAndArgs[0].(string), messageAndArgs[1:]...)
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

// TODO: line and column number not implemented
func HaveZeroOrOneValues(validator V.Validator, _ Field, values []string) *V.ValidationError {
	if len(values) > 1 {
		return V.Err(validator.Name(), V.InvalidValue, 0, 0, "should have exactly zero or one value")
	}

	return nil
}

func WhenFieldEquals(conditionField Field, conditionValues ...string) RulePredicate {
	return func(validator V.Validator, unit P.UnitFile, field Field) bool {
		values := unit.LookupAll(conditionField.Group, conditionField.Key)
		for _, fieldValue := range values {
			for _, conditionValue := range conditionValues {
				if fieldValue.Value == conditionValue {
					return true
				}
			}
		}
		return false
	}
}

func Always(_ V.Validator, _ P.UnitFile, _ Field) bool {
	return true
}

func MatchRegexp(regexp regexp.Regexp) ValuesValidator {
	return func(validator V.Validator, field Field, values []P.UnitValue) *V.ValidationError {
		for _, value := range values {
			if !regexp.MatchString(value.Value) {
				return V.Err(validator.Name(), V.InvalidValue, 0, 0,
					fmt.Sprintf("Must match regexp '%s'", regexp.String()))
			}
		}
		return nil
	}
}
