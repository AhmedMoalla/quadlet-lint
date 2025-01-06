package rules

import (
	"regexp"
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/container"
	P "github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/stretchr/testify/assert"
)

var v = newTestValidator(V.Options{})

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
		{"NoErrorsIfFieldAbsent", "[Container]\nNetwork=test", 0, -1},
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
	vRef := newTestValidator(V.Options{CheckReferences: true}, "test.network", "test.container")
	tests := []struct {
		name      string
		unit      string
		validator V.Validator
		errors    []V.Location
	}{
		{"NoErrorsIfRefCheckDisabled", "[Container]\nNetwork=bad.container", v, nil},
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=5", vRef, nil},
		{"ReferencesCorrectly", "[Container]\nNetwork=test.network", vRef, nil},
		{"ReferencesCorrectly2", "[Container]\nNetwork=test.container", vRef, nil},
		{"BadReference", "[Container]\nNetwork=bad.container", vRef, []V.Location{{"", 2, 8}}},
		{"BadReferences", "[Container]\nNetwork=bad.container\nOther=6\nNetwork=otherbad.network", vRef,
			[]V.Location{{"", 2, 8}, {"", 4, 8}}},
	}

	rule := CanReference(P.UnitTypeNetwork, P.UnitTypeContainer)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(test.validator, unit, container.Network)
			assert.Len(t, errs, len(test.errors))

			if len(errs) > 0 {
				for i, err := range errs {
					assert.Equal(t, test.validator.Name(), err.ValidatorName)
					assert.Equal(t, V.InvalidReference, err.ErrorType)
					assert.Equal(t, test.errors[i].Line, err.Line)
					assert.Equal(t, test.errors[i].Column, err.Column)
				}
			}
		})
	}
}

func TestHaveFormat(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldWellFormatted", "[Container]\nNetwork=test", nil},
		{"FieldWellFormatted2", "[Container]\nNetwork=test:opt1=val1", nil},
		{"FieldWellFormatted3", "[Container]\nNetwork=test:opt1=val1,opt2=val2", nil},
		{"FieldWellFormatted4", "[Container]\nNetwork=test:opt1", nil},
		{"FieldBadFormat", "[Container]\nNetwork=test:", []V.Location{{"", 2, 8}}},
		{"FieldBadFormat2", "[Container]\nNetwork=test:opt1,", []V.Location{{"", 2, 8}}},
		{"FieldBadFormat3", "[Container]\nNetwork=test:test2:test3", []V.Location{{"", 2, 8}}},
	}

	format := Format{Name: "TestFormat", ValueSeparator: ":", OptionsSeparator: ","}
	rule := HaveFormat(format)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(v, unit, container.Network)
			assert.Len(t, errs, len(test.errors))

			if len(errs) > 0 {
				for i, err := range errs {
					assert.Equal(t, v.Name(), err.ValidatorName)
					assert.Equal(t, V.InvalidValue, err.ErrorType)
					assert.Equal(t, test.errors[i].Line, err.Line)
					assert.Equal(t, test.errors[i].Column, err.Column)
				}
			}
		})
	}
}

func TestAllowedValues(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldHasSomeAllowedValues", "[Container]\nPublishPort=val1\nPublishPort=val2", nil},
		{"FieldHasAllAllowedValues", "[Container]\nPublishPort=val1\nPublishPort=val2\nPublishPort=val3", nil},
		{"FieldHasBadValues", "[Container]\nPublishPort=bad\nPublishPort=bad2",
			[]V.Location{{"", 2, 12}, {"", 3, 12}}},
		{"FieldHasSomeBadValues", "[Container]\nPublishPort=bad\nOther=test\nPublishPort=val2\nPublishPort=bad2",
			[]V.Location{{"", 2, 12}, {"", 5, 12}}},
	}

	rule := AllowedValues("val1", "val2", "val3")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(v, unit, container.PublishPort)
			assert.Len(t, errs, len(test.errors))

			if len(errs) > 0 {
				for i, err := range errs {
					assert.Equal(t, v.Name(), err.ValidatorName)
					assert.Equal(t, V.InvalidValue, err.ErrorType)
					assert.Equal(t, test.errors[i].Line, err.Line)
					assert.Equal(t, test.errors[i].Column, err.Column)
				}
			}
		})
	}
}

func TestHasSuffix(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldWithSuffix", "[Container]\nPod=test.pod", nil},
		{"FieldWithoutSuffix", "[Container]\nPod=bad", []V.Location{{"", 2, 4}}},
	}

	rule := HasSuffix(".pod")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(v, unit, container.Pod)
			assert.Len(t, errs, len(test.errors))

			if len(errs) > 0 {
				for i, err := range errs {
					assert.Equal(t, v.Name(), err.ValidatorName)
					assert.Equal(t, V.InvalidValue, err.ErrorType)
					assert.Equal(t, test.errors[i].Line, err.Line)
					assert.Equal(t, test.errors[i].Column, err.Column)
				}
			}
		})
	}
}

func TestDependsOn(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldWithDependency", "[Container]\nUser=1\nGroup=1", nil},
		{"FieldWithoutDependency", "[Container]\nGroup=1", []V.Location{{"", 2, 0}}},
	}

	rule := DependsOn(container.User)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(v, unit, container.Group)
			assert.Len(t, errs, len(test.errors))

			if len(errs) > 0 {
				for i, err := range errs {
					assert.Equal(t, v.Name(), err.ValidatorName)
					assert.Equal(t, V.UnsatisfiedDependency, err.ErrorType)
					assert.Equal(t, test.errors[i].Line, err.Line)
					assert.Equal(t, test.errors[i].Column, err.Column)
				}
			}
		})
	}
}

func TestDeprecated(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"DeprecatedFieldPresent", "[Container]\nRemapUid=1\nOther=1\nRemapUid=2",
			[]V.Location{{"", 2, 0}, {"", 4, 0}}},
	}

	rule := Deprecated
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(v, unit, container.RemapUid)
			assert.Len(t, errs, len(test.errors))

			if len(errs) > 0 {
				for i, err := range errs {
					assert.Equal(t, v.Name(), err.ValidatorName)
					assert.Equal(t, V.DeprecatedKey, err.ErrorType)
					assert.Equal(t, test.errors[i].Line, err.Line)
					assert.Equal(t, test.errors[i].Column, err.Column)
				}
			}
		})
	}
}

func TestMatchRegexp(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldMatchesRegexp", "[Container]\nRemapUid=1\nOther=1\nRemapUid=2", nil},
		{"FieldDoesNotMatchRegexp", "[Container]\nRemapUid=1\nOther=1\nRemapUid=abcd", []V.Location{{"", 4, 9}}},
	}

	rule := MatchRegexp(regexp.MustCompile(`\d+`))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(v, unit, container.RemapUid)
			assert.Len(t, errs, len(test.errors))

			if len(errs) > 0 {
				for i, err := range errs {
					assert.Equal(t, v.Name(), err.ValidatorName)
					assert.Equal(t, V.InvalidValue, err.ErrorType)
					assert.Equal(t, test.errors[i].Line, err.Line)
					assert.Equal(t, test.errors[i].Column, err.Column)
				}
			}
		})
	}
}

func TestValuesMust(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldMatchesCondition", "[Container]\nRemapUid=1\nUser=test", nil},
		{"FieldMatchesCondition2", "[Container]\nUser=test", nil},
		{"NoErrorsIfRulePredicateIsFalse", "[Container]\nRemapUid=1\nRemapUid=2\nUser=other", nil},
		{"ConditionNotRespected", "[Container]\nRemapUid=1\nRemapUid=2\nUser=test", []V.Location{{"", 2, 9}}},
	}

	rule := ValuesMust(HaveZeroOrOneValues, WhenFieldEquals(container.User, "test"),
		"field %s should have zero or one value when field %s have value 'test'", container.RemapUid.Key, container.User.Key)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			errs := rule(v, unit, container.RemapUid)
			assert.Len(t, errs, len(test.errors))

			if len(errs) > 0 {
				for i, err := range errs {
					assert.Equal(t, v.Name(), err.ValidatorName)
					assert.Equal(t, V.InvalidValue, err.ErrorType)
					assert.Equal(t, test.errors[i].Line, err.Line)
					assert.Equal(t, test.errors[i].Column, err.Column)
				}
			}
		})
	}
}

func TestHaveZeroOrOneValues(t *testing.T) {
	tests := []struct {
		name  string
		unit  string
		error *V.Location
	}{
		{"HasOneValue", "[Container]\nRemapUid=1", nil},
		{"HasZeroValues", "[Container]\nOther=test", nil},
		{"HasManyValues", "[Container]\nRemapUid=1\nRemapUid=1", &V.Location{Line: 3, Column: 9}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			res, _ := unit.Lookup(container.RemapUid)

			if err := HaveZeroOrOneValues(v, container.RemapUid, res.Values); err != nil {
				assert.Equal(t, v.Name(), err.ValidatorName)
				assert.Equal(t, V.InvalidValue, err.ErrorType)
				if test.error != nil {
					assert.Equal(t, test.error.Line, err.Line)
					assert.Equal(t, test.error.Column, err.Column)
				}
			}
		})
	}
}

func TestWhenFieldEquals(t *testing.T) {
	tests := []struct {
		name   string
		unit   string
		result bool
	}{
		{"FieldHasValue", "[Container]\nUser=val1", true},
		{"FieldHasValue2", "[Container]\nUser=val2", true},
		{"FieldAbsent", "[Container]\nOther=5", false},
		{"FieldHasBadValue", "[Container]\nUser=test", false},
	}

	rule := WhenFieldEquals(container.User, "val1", "val2")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := parseString(t, test.unit)
			result := rule(v, unit, container.User)
			assert.Equal(t, test.result, result)
		})
	}
}

func parseString(t *testing.T, content string) P.UnitFile {
	t.Helper()
	unit, errs := P.ParseUnitFileString("test.container", content)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Error(err.Error())
		}
		t.Fatal("errors while parsing file content")
	}

	return *unit
}

type testValidator struct {
	ctx V.Context
}

func (t testValidator) Name() string {
	return "test"
}

func (t testValidator) Context() V.Context {
	return t.ctx
}

func (t testValidator) Validate(_ P.UnitFile) []V.ValidationError {
	return nil
}

func newTestValidator(options V.Options, files ...string) V.Validator {
	units := make([]P.UnitFile, 0, len(files))
	for _, file := range files {
		units = append(units, P.UnitFile{Filename: file})
	}
	return testValidator{ctx: V.Context{
		Options:      options,
		AllUnitFiles: units,
	}}
}
