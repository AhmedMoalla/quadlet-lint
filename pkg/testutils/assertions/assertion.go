package assertions

import (
	"fmt"
	"slices"
	"testing"

	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/stretchr/testify/assert"
)

type AssertionType string

const (
	AssertError AssertionType = "assert-error"
)

func ParseAssertionType(assertion string) (AssertionType, error) {
	if AssertionType(assertion) == AssertError {
		return AssertError, nil
	}

	return "", fmt.Errorf("assertion type '%s' not recognized", assertion)
}

type Assertion struct {
	Type        AssertionType
	ErrCategory string
	ErrName     string
	Group       string
	Key         string
	Line        int
	Column      int
}

type Assertions []Assertion

func (a Assertions) RunAssertions(t *testing.T, errors []V.ValidationError) {
	t.Helper()

	if len(a) == 0 {
		assert.Empty(t, errors)
		return
	}

	for _, assertion := range a {
		switch assertion.Type {
		case AssertError:
			result := slices.ContainsFunc(errors, matchesAssertion(assertion))

			if !result {
				t.Errorf("expected '%s%s' error concerning key '%s.%s' on line %d column %d but no errors were found.",
					assertion.ErrCategory, "."+assertion.ErrName, assertion.Group, assertion.Key, assertion.Line, assertion.Column)
			} else {
				errors = slices.DeleteFunc(errors, matchesAssertion(assertion))
			}
		default:
			panic(fmt.Sprintf("assertion type '%s' not recognized", assertion.Type))
		}
	}

	if len(errors) > 0 {
		t.Errorf("Some errors were not asserted: ")
		for _, err := range errors {
			t.Errorf("\t- %s\n", err)
		}
	}

	if t.Failed() {
		t.Errorf("assertions failed for errors")
		for _, err := range errors {
			t.Errorf("\t- %v for Key: '%s.%s' at Line: %d, Column: %d\n => %s",
				err, err.Group, err.Key, err.Line, err.Column, err.Error)
		}
	}
}

func matchesAssertion(assertion Assertion) func(err V.ValidationError) bool {
	return func(err V.ValidationError) bool {
		return (assertion.ErrName == "" || err.ErrorName == assertion.ErrName) &&
			err.ErrorCategory.Name == assertion.ErrCategory &&
			err.Group == assertion.Group &&
			err.Key == assertion.Key &&
			err.Line == assertion.Line &&
			err.Column == assertion.Column
	}
}
