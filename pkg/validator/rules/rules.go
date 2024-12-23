package rules

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	M "github.com/AhmedMoalla/quadlet-lint/pkg/generated/model"
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
func CheckRules(validator V.Validator, unit P.UnitFile, rules M.Groups) []V.ValidationError {
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
				field, ok := M.Fields[groupName][fieldName]
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

func RequiredIfNotPresent(other P.Field) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field P.Field) []V.ValidationError {
		if !unit.HasValue(other.Group, other.Key) && !unit.HasValue(field.Group, field.Key) {
			return ErrSlice(validator.Name(), V.RequiredKey, 0, 0,
				fmt.Sprintf("at least one of these keys is required: %s, %s", field, other))
		}

		return nil
	}
}

func ConflictsWith(others ...P.Field) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field P.Field) []V.ValidationError {
		validationErrors := make([]V.ValidationError, 0)
		for _, other := range others {
			if unit.HasValue(other.Group, other.Key) && unit.HasValue(field.Group, field.Key) {
				validationErrors = append(validationErrors, *V.Err(validator.Name(), V.KeyConflict, 0, 0,
					fmt.Sprintf("the keys %s, %s cannot be specified together", field, other)))
			}
		}

		return validationErrors
	}
}

func CanReference(unitTypes ...P.UnitType) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field P.Field) []V.ValidationError {
		context := validator.Context()
		if !context.CheckReferences {
			return nil
		}

		var values []string
		if field.Multiple {
			values = unit.LookupAll(field.Group, field.Key)
		} else if value, found := unit.Lookup(field.Group, field.Key); found {
			values = []string{value}
		}

		if len(values) == 0 {
			return nil
		}

		units := context.AllUnitFiles
		validationErrors := make([]V.ValidationError, 0)
		for _, value := range values {
			for _, unitType := range unitTypes {
				if strings.HasSuffix(value, string("."+unitType)) { // TODO: Add extension as field to UnitType
					foundUnit := slices.ContainsFunc(units, func(unit P.UnitFile) bool {
						return unit.Filename == value
					})

					if !foundUnit {
						validationErrors = append(validationErrors, *V.Err(validator.Name(), V.InvalidReference, 0, 0,
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
	return func(validator V.Validator, field P.Field, values []string) *V.ValidationError {
		for _, value := range values {
			err := format.ParseAndValidate(value)
			if err != nil {
				return V.Err(validator.Name(), V.InvalidValue, 0, 0, err.Error())
			}
		}

		return nil
	}
}

func AllowedValues(allowedValues ...string) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field P.Field) []V.ValidationError {
		value, ok := unit.Lookup(field.Group, field.Key)
		if ok && !slices.Contains(allowedValues, value) {
			return ErrSlice(validator.Name(), V.InvalidValue, 0, 0,
				fmt.Sprintf("invalid value '%s' for key '%s'. Allowed values: %s",
					value, field, allowedValues))
		}
		return nil
	}
}

func HasSuffix(suffix string) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field P.Field) []V.ValidationError {
		value, found := unit.Lookup(field.Group, field.Key)
		if !found {
			return nil
		}

		if !strings.HasSuffix(value, suffix) {
			return ErrSlice(validator.Name(), V.InvalidValue, 0, 0,
				fmt.Sprintf("value '%s' must have suffix '%s'", value, suffix))
		}

		return nil
	}
}

func DependsOn(dependency P.Field) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field P.Field) []V.ValidationError {
		dependency, dependencyFound := unit.Lookup(dependency.Group, dependency.Key)
		dependencyOk := dependencyFound && len(dependency) > 0

		value, found := unit.Lookup(field.Group, field.Key)
		fieldOk := found && len(value) > 0

		if !dependencyOk && fieldOk {
			return ErrSlice(validator.Name(), V.UnsatisfiedDependency, 0, 0,
				fmt.Sprintf("value for '%s' was set but it depends on key '%s' which was not found",
					field, dependency))
		}

		return nil
	}
}

func Deprecated(validator V.Validator, unit P.UnitFile, field P.Field) []V.ValidationError {
	if _, found := unit.Lookup(field.Group, field.Key); found {
		return ErrSlice(validator.Name(), V.DeprecatedKey, 0, 0,
			fmt.Sprintf("key '%s' is deprecated and should not be used", field))
	}

	return nil
}

func ValuesMust(valuePredicate ValuesValidator, rulePredicate RulePredicate, messageAndArgs ...any) V.Rule {
	return func(validator V.Validator, unit P.UnitFile, field P.Field) []V.ValidationError {
		if rulePredicate(validator, unit, field) {
			// TODO: Should use correct Lookup function depending on the field
			// Refactor Lookup function to take Field instances
			// Fields should define LookupMode property that tells which Lookup function to use
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

type ValuesValidator func(validator V.Validator, field P.Field, values []string) *V.ValidationError
type RulePredicate func(validator V.Validator, unit P.UnitFile, field P.Field) bool

func HaveZeroOrOneValues(validator V.Validator, _ P.Field, values []string) *V.ValidationError {
	if len(values) > 1 {
		return V.Err(validator.Name(), V.InvalidValue, 0, 0, "should have exactly zero or one value")
	}

	return nil
}

func WhenFieldEquals(conditionField P.Field, conditionValues ...string) RulePredicate {
	return func(validator V.Validator, unit P.UnitFile, field P.Field) bool {
		values := unit.LookupAll(conditionField.Group, conditionField.Key)
		for _, fieldValue := range values {
			for _, conditionValue := range conditionValues {
				if fieldValue == conditionValue {
					return true
				}
			}
		}
		return false
	}
}

func Always(_ V.Validator, _ P.UnitFile, _ P.Field) bool {
	return true
}

func MatchRegexp(regexp regexp.Regexp) ValuesValidator {
	return func(validator V.Validator, field P.Field, values []string) *V.ValidationError {
		for _, value := range values {
			if !regexp.MatchString(value) {
				return V.Err(validator.Name(), V.InvalidValue, 0, 0,
					fmt.Sprintf("Must match regexp '%s'", regexp.String()))
			}
		}
		return nil
	}
}
