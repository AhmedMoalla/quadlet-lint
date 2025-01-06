package rules

import (
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/container"
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/stretchr/testify/assert"
)

type testValidator struct {
	ctx V.Context
}

func (t testValidator) Name() string {
	return "test"
}

func (t testValidator) Context() V.Context {
	return t.ctx
}

func (t testValidator) Validate(_ parser.UnitFile) []V.ValidationError {
	return nil
}

var v = newTestValidator(V.Options{})

func newTestValidator(options V.Options, files ...string) V.Validator {
	units := make([]parser.UnitFile, 0, len(files))
	for _, file := range files {
		units = append(units, parser.UnitFile{Filename: file})
	}
	return testValidator{ctx: V.Context{
		Options:      options,
		AllUnitFiles: units,
	}}
}

func TestRequiredIfNotPresent(t *testing.T) {
	tests := []struct {
		name    string
		unit    string
		nErrors int
	}{
		{"BothFieldsAbsent", "[Container]\nNetwork=test", 1},
		{"TargetAbsentButFieldPresent", "[Container]\nRootfs=test", 0},
		{"TargetPresentButFieldAbsent", "[Container]\nImage=test", 0},
	}

	rule := RequiredIfNotPresent(container.Image)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(v, unit, container.Rootfs)

			assert.Len(t, errs, test.nErrors)

			if len(errs) > 0 {
				for _, err := range errs {
					assert.Equal(t, v.Name(), err.ValidatorName)
					assert.Equal(t, V.RequiredKey, err.ErrorType)
					assert.Equal(t, 0, err.Line)
					assert.Equal(t, 0, err.Column)
				}
			}
		})
	}
}

func TestConflictsWith(t *testing.T) {
	tests := []struct {
		name      string
		unit      string
		nErrors   int
		errorLine int
	}{
		{"FieldAbsent", "[Container]\nNetwork=test", 0, -1},
		{"OnlyFieldPresent", "[Container]\nRemapUid=1", 0, -1},
		{"FieldPresentWithOneConflictingTarget", "[Container]\nRemapUid=1\nUserNS=1", 1, 2},
		{"FieldPresentWithAllConflictingTargets", "[Container]\nUIDMap=1\nRemapUid=1\nUserNS=test", 2, 3},
	}

	rule := ConflictsWith(container.UserNS, container.UIDMap)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(v, unit, container.RemapUid)
			assert.Len(t, errs, test.nErrors)

			if len(errs) > 0 {
				for _, err := range errs {
					assert.Equal(t, v.Name(), err.ValidatorName)
					assert.Equal(t, V.KeyConflict, err.ErrorType)
					assert.Equal(t, test.errorLine, err.Line)
					assert.Equal(t, 0, err.Column)
				}
			}
		})
	}
}

func TestCanReference(t *testing.T) {
	vRef := newTestValidator(V.Options{CheckReferences: true}, "test.pod", "test.container")
	tests := []struct {
		name      string
		unit      string
		nErrors   int
		locations []V.Location
		validator V.Validator
	}{
		{"NoErrorsIfRefCheckDisabled", "[Container]\nNetwork=bad.container", 0, nil, v},
		{"ReferencesCorrectly", "[Container]\nNetwork=test.pod", 0, nil, vRef},
		{"ReferencesCorrectly2", "[Container]\nNetwork=test.container", 0, nil, vRef},
		{"BadReference", "[Container]\nNetwork=bad.container", 1, []V.Location{{"", 2, 8}}, vRef},
		{"BadReferences", "[Container]\nNetwork=bad.container\nOther=6\nNetwork=otherbad.pod", 2,
			[]V.Location{{"", 2, 8}, {"", 4, 8}}, vRef},
	}

	rule := CanReference(parser.UnitTypePod, parser.UnitTypeContainer)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(test.validator, unit, container.Network)
			assert.Len(t, errs, test.nErrors)
		})
	}
}

func parseString(t *testing.T, content string) parser.UnitFile {
	t.Helper()
	unit, errs := parser.ParseUnitFileString("test.container", content)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Error(err.Error())
		}
		t.Fatal("errors while parsing file content")
	}

	return *unit
}
