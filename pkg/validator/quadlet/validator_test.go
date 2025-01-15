package quadlet

import (
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model"
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerValidator_Validate(t *testing.T) {
	errors := validate(t, "testdata/unit.container")
	assert.Empty(t, errors)

	errors = validate(t, "testdata/err.container")
	assert.NotEmpty(t, errors)
	// TODO: implement ## assert-error RequiredKey Container Image 0 0
}

func validate(t *testing.T, filename string) []V.ValidationError {
	unit, err := parser.ParseUnitFile(filename)
	require.Empty(t, err)
	return Validator([]model.UnitFile{unit}, V.Options{CheckReferences: true}).Validate(unit)
}
