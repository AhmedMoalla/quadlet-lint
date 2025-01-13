package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	errLevel = []ValidationError{
		{Message: "Error1", ErrorType: ErrorType{Level: LevelError}},
		{Message: "Error2", ErrorType: ErrorType{Level: LevelError}},
		{Message: "Error3", ErrorType: ErrorType{Level: LevelError}},
	}
	warnLevel = []ValidationError{
		{Message: "Warn1", ErrorType: ErrorType{Level: LevelWarning}},
		{Message: "Warn2", ErrorType: ErrorType{Level: LevelWarning}},
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
	errs["test.go"] = append(errs["test.go"], ValidationError{Message: "Other", ErrorType: ErrorType{Level: "Other"}})
	assert.False(t, errs.HasErrors())
}

func TestValidationErrors_AddError(t *testing.T) {
	t.Parallel()

	errs := make(ValidationErrors)
	errs["test.go"] = append(errs["test.go"], errLevel...)

	assert.Len(t, errs["test.go"], len(errLevel))

	errs.AddError("test.go", ValidationError{Message: "Other", ErrorType: ErrorType{Level: "Other"}})
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
