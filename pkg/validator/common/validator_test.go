package common

import (
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/testutils"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Parallel()

	unit := testutils.ParseString(t, unitFileToTest)

	errs := validator.Validate(unit)
	require.Len(t, errs, 3)

	expectedErrLines := []int{4, 7, 8}
	for i, err := range errs {
		assertUnknownKeyError(t, err, expectedErrLines[i])
	}
}

func assertUnknownKeyError(t *testing.T, err V.ValidationError, line int) {
	t.Helper()

	assert.Equal(t, validator.Name(), err.ValidatorName)
	assert.Equal(t, V.UnknownKey, err.ErrorCategory)
	assert.Equal(t, 0, err.Column)
	assert.Equal(t, line, err.Line, err.Error)
}
