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

func TestCommonValidator_Validate(t *testing.T) {
	unit := testutils.ParseString(t, unitFileToTest)

	validator := Validator()
	errs := validator.Validate(unit)
	assert.Len(t, errs, 3)

	err := errs[0]
	assert.Equal(t, validator.Name(), err.ValidatorName)
	assert.Equal(t, V.UnknownKey, err.ErrorType)
	assert.Equal(t, 0, err.Column)
	assert.Equal(t, 4, err.Line, err.Message)

	err = errs[1]
	assert.Equal(t, validator.Name(), err.ValidatorName)
	assert.Equal(t, V.UnknownKey, err.ErrorType)
	assert.Equal(t, 0, err.Column)
	assert.Equal(t, 7, err.Line, err.Message)

	err = errs[2]
	assert.Equal(t, validator.Name(), err.ValidatorName)
	assert.Equal(t, V.UnknownKey, err.ErrorType)
	assert.Equal(t, 0, err.Column)
	assert.Equal(t, 8, err.Line, err.Message)
}
