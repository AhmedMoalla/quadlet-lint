package validator

import (
	"fmt"
	"slices"
	"strings"

	"github.com/containers/storage/pkg/regexp"

	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
)

type CheckerFn func(validator Validator, unit parser.UnitFile) []ValidationError
type PredicateFn func(value string) bool
type Predicate struct {
	fn             PredicateFn
	message        string
	negatedMessage string
	negated        bool
}

func (p *Predicate) msg(value string) string {
	if p.negated {
		return fmt.Sprintf(p.negatedMessage, value)
	}
	return fmt.Sprintf(p.message, value)
}

func (p *Predicate) Negate() *Predicate {
	return &Predicate{
		fn: func(value string) bool {
			return !p.fn(value)
		},
		message:        p.message,
		negatedMessage: p.negatedMessage,
		negated:        !p.negated,
	}
}

func (p *Predicate) And(other *Predicate) *Predicate {
	return &Predicate{
		fn: func(value string) bool {
			return p.fn(value) && other.fn(value)
		},
		// TODO: Fix messages as these are going to be broken when multiple fmt.Sprintf placeholders are present due to
		// joining the messages
		message:        strings.Join([]string{p.message, other.message}, ", "),
		negatedMessage: strings.Join([]string{p.negatedMessage, other.negatedMessage}, ", "),
	}
}

func ErrSlice(validatorName string, errType ErrorType, line, column int, message string) []ValidationError {
	return []ValidationError{*Err(validatorName, errType, line, column, message)}
}

func DoChecks(validator Validator, unit parser.UnitFile, checkers ...CheckerFn) []ValidationError {
	validationErrors := make([]ValidationError, 0, len(checkers))
	for _, checker := range checkers {
		err := checker(validator, unit)
		if err != nil {
			validationErrors = append(validationErrors, err...)
		}
	}
	return validationErrors
}

func CheckForRequiredKey(groupName string, requiredKeyCandidates ...string) CheckerFn {
	return func(validator Validator, unit parser.UnitFile) []ValidationError {
		fmt.Println("CheckForRequiredKey:", unit.Filename, groupName, requiredKeyCandidates)
		for _, key := range requiredKeyCandidates {
			if value, _ := unit.Lookup(groupName, key); len(value) > 0 {
				return nil
			}
		}
		return ErrSlice(validator.Name(), RequiredKey, 0, 0,
			fmt.Sprintf("at least one of these keys is required: %s", requiredKeyCandidates))
	}
}

func CheckForKeyConflict(groupName string, conflictingKeys ...string) CheckerFn {
	return func(validator Validator, unit parser.UnitFile) []ValidationError {
		fmt.Println("CheckForKeyConflict:", unit.Filename, groupName, conflictingKeys)
		keysFound := make([]string, 0, len(conflictingKeys))
		for _, key := range conflictingKeys {
			if value, _ := unit.Lookup(groupName, key); len(value) > 0 {
				keysFound = append(keysFound, key)
			}
		}

		if len(keysFound) <= 1 {
			return nil
		}

		return ErrSlice(validator.Name(), KeyConflict, 0, 0,
			fmt.Sprintf("the keys %s cannot be specified together", keysFound))
	}
}

func CheckForAllowedValues(groupName string, key string, allowedValues ...string) CheckerFn {
	return func(validator Validator, unit parser.UnitFile) []ValidationError {
		fmt.Println("CheckForAllowedValues:", unit.Filename, groupName, key, allowedValues)
		value, ok := unit.Lookup(groupName, key)
		if ok && !slices.Contains(allowedValues, value) {
			return ErrSlice(validator.Name(), InvalidValue, 0, 0,
				fmt.Sprintf("invalid value '%s' for key '[%s]%s'. Allowed values: %s",
					value, groupName, key, allowedValues))
		}
		return nil
	}
}

func CheckForUnknownKeys(groupName string, supportedKeys map[string]bool) CheckerFn {
	return func(validator Validator, unit parser.UnitFile) []ValidationError {
		fmt.Println("CheckForUnknownKeys:", unit.Filename, groupName)
		keys := unit.ListKeys(groupName)
		for _, key := range keys {
			if !supportedKeys[key] {
				return ErrSlice(validator.Name(), UnknownKey, 0, 0,
					fmt.Sprintf("unsupported key '%s' in group '%s' in %s", key, groupName, unit.Path))
			}
		}

		return nil
	}
}

func CheckForInvalidValuesWithPredicateFn(groupName string, key string, predicate PredicateFn, message string, args ...any) CheckerFn {
	return CheckForInvalidValuesWithMessage(groupName, key, &Predicate{fn: predicate}, message, args...)
}

func CheckForInvalidValues(groupName string, key string, predicate *Predicate) CheckerFn {
	return checkForInvalidValues(groupName, key, predicate, func(value string) string {
		return predicate.msg(value)
	})
}

func CheckForInvalidValuesWithMessage(groupName string, key string, predicate *Predicate, message string, args ...any) CheckerFn {
	return checkForInvalidValues(groupName, key, predicate, func(value string) string {
		return fmt.Sprintf(strings.Replace(message, "{value}", value, 1), args...)
	})
}

func checkForInvalidValues(groupName string, key string, predicate *Predicate, message func(value string) string) CheckerFn {
	return func(validator Validator, unit parser.UnitFile) []ValidationError {
		fmt.Println("checkForInvalidValues:", unit.Filename, groupName, key)
		values := unit.LookupAll(groupName, key)
		for _, value := range values {
			value := strings.TrimSpace(value)
			if predicate.fn(value) {
				// TODO: Make CheckerFn return list of ValidationErrors and return all of them here
				return ErrSlice(validator.Name(), InvalidValue, 0, 0, message(value))
			}
		}
		return nil
	}
}

func CheckForInvalidValue(groupName string, key string, predicate *Predicate) CheckerFn {
	return checkForInvalidValue(groupName, key, predicate, func(value string) string {
		return predicate.msg(value)
	})
}

func CheckForInvalidValueWithMessage(groupName string, key string, predicate *Predicate, message string, args ...string) CheckerFn {
	return checkForInvalidValue(groupName, key, predicate, func(value string) string {
		return fmt.Sprintf(strings.Replace(message, "{value}", value, 1), args)
	})
}

func checkForInvalidValue(groupName string, key string, predicate *Predicate, message func(value string) string) CheckerFn {
	return func(validator Validator, unit parser.UnitFile) []ValidationError {
		fmt.Println("checkForInvalidValue:", unit.Filename, groupName, key)
		value, ok := unit.Lookup(groupName, key)
		if ok && predicate.fn(value) {
			return ErrSlice(validator.Name(), InvalidValue, 0, 0, message(value))
		}
		return nil
	}
}

// ==================== Predicates ====================

func HasLength() *Predicate {
	return &Predicate{
		fn: func(value string) bool {
			return len(value) > 0
		},
		message:        "%s is not empty",
		negatedMessage: "%s is empty",
	}
}

func HasSuffix(suffix string) *Predicate {
	return &Predicate{
		fn: func(value string) bool {
			return strings.HasSuffix(value, suffix)
		},
		message:        fmt.Sprintf("%%s has suffix: %s", suffix),
		negatedMessage: fmt.Sprintf("%%s does not have suffix: %s", suffix),
	}
}

func MatchesRegex(regex regexp.Regexp) *Predicate {
	return &Predicate{
		fn: func(value string) bool {
			return regex.MatchString(value)
		},
		message:        fmt.Sprintf("%%s matches regex: %s", regex),
		negatedMessage: fmt.Sprintf("%%s does not match regex: %s", regex),
	}
}
