package common

import (
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/testutils"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/stretchr/testify/assert"
)

const unitFileToTest = `[Container]
ContainerName=my-container
# Unknown key
Test=bla

[Pod]
Name=my-pod
Other=bla

# Ignored group
[Service]
dazdaz=dadazdazd
`

var validator = Validator()

func TestCommonValidator_Validate(t *testing.T) {
	unit := testutils.ParseString(t, unitFileToTest)

	errs := validator.Validate(unit)
	assert.Len(t, errs, 3)

	expectedErrLines := []int{4, 7, 8}
	for i, err := range errs {
		assertUnknownKeyError(t, err, expectedErrLines[i])
	}
}

func assertUnknownKeyError(t *testing.T, err V.ValidationError, line int) {
	assert.Equal(t, validator.Name(), err.ValidatorName)
	assert.Equal(t, V.UnknownKey, err.ErrorType)
	assert.Equal(t, 0, err.Column)
	assert.Equal(t, line, err.Line, err.Message)
}
