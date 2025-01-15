package validator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	errOther = errors.New("other")

	errLevel = []ValidationError{
		{Error: errors.New("error1"), ErrorCategory: ErrorCategory{Level: LevelError}},
		{Error: errors.New("error2"), ErrorCategory: ErrorCategory{Level: LevelError}},
		{Error: errors.New("error3"), ErrorCategory: ErrorCategory{Level: LevelError}},
	}
	warnLevel = []ValidationError{
		{Error: errors.New("warn1"), ErrorCategory: ErrorCategory{Level: LevelWarning}},
		{Error: errors.New("warn2"), ErrorCategory: ErrorCategory{Level: LevelWarning}},
	}
)

func TestValidationErrors_WhereLevel(t *testing.T) {
	t.Parallel()

	errs := make(ValidationErrors)
	errs["test.go"] = append(errLevel, warnLevel...)

	assert.ElementsMatch(t, errLevel, errs.WhereLevel(LevelError))
	assert.ElementsMatch(t, warnLevel, errs.WhereLevel(LevelWarning))
}

func TestValidationErrors_HasErrors(t *testing.T) {
	t.Parallel()

	errs := make(ValidationErrors)
	errs["test.go"] = append(errLevel, warnLevel...)

	assert.True(t, errs.HasErrors())

	errs = make(ValidationErrors)
	assert.False(t, errs.HasErrors())

	errs = make(ValidationErrors)
	errs["test.go"] = append(errs["test.go"], ValidationError{
		Error:         errOther,
		ErrorCategory: ErrorCategory{Level: "Other"},
	})
	assert.False(t, errs.HasErrors())
}

func TestValidationErrors_AddError(t *testing.T) {
	t.Parallel()

	errs := make(ValidationErrors)
	errs["test.go"] = append(errs["test.go"], errLevel...)

	assert.Len(t, errs["test.go"], len(errLevel))

	errs.AddError("test.go", ValidationError{Error: errOther, ErrorCategory: ErrorCategory{Level: "Other"}})
	assert.Len(t, errs["test.go"], len(errLevel)+1)
}

func TestValidationErrors_Merge(t *testing.T) {
	t.Parallel()

	errs1 := make(ValidationErrors)
	errs1.AddError("test.go", errLevel...)

	errs2 := make(ValidationErrors)
	errs2.AddError("test.go", warnLevel...)

	errs := errs1.Merge(errs2)
	assert.ElementsMatch(t, append(errLevel, warnLevel...), errs["test.go"])
	assert.ElementsMatch(t, append(errLevel, warnLevel...), errs1["test.go"])
	assert.ElementsMatch(t, warnLevel, errs2["test.go"])
}
