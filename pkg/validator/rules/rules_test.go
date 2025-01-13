package rules

import (
	"regexp"
	"slices"
	"testing"

	M "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	model "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/container"
	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/service"
	"github.com/AhmedMoalla/quadlet-lint/pkg/testutils"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/stretchr/testify/assert"
)

var v = testutils.NewTestValidator(V.Options{})

func TestCheckRules(t *testing.T) {
	t.Parallel()

	unit := testutils.ParseString(t, "[Container]\nOther=test\n[Service]\nKillMode=bad")
	rules := model.Groups{
		Container: container.GContainer{
			Rootfs: Rules(RequiredIfNotPresent(container.Image)),
		},
		Service: service.GService{
			KillMode: Rules(AllowedValues("mixed", "control-group")),
		},
	}

	errs := CheckRules(v, unit, rules)
	assert.Len(t, errs, 2)

	assert.True(t, slices.ContainsFunc(errs, func(err V.ValidationError) bool {
		return err.ValidatorName == v.Name() &&
			err.ErrorType == V.RequiredKey &&
			err.Line == 0 && err.Column == 0
	}))

	assert.True(t, slices.ContainsFunc(errs, func(err V.ValidationError) bool {
		return err.ValidatorName == v.Name() &&
			err.ErrorType == V.InvalidValue &&
			err.Line == 4 && err.Column == 9
	}))
}

func TestCheckRulesShouldPanicIfFieldNotGeneratedInModel(t *testing.T) {
	t.Parallel()

	field := model.Fields["Container"][container.Rootfs.Key]
	delete(model.Fields["Container"], container.Rootfs.Key)
	assert.Panics(t, func() {
		TestCheckRules(t)
	})
	model.Fields["Container"][container.Rootfs.Key] = field
}

func TestRequiredIfNotPresent(t *testing.T) {
	t.Parallel()

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

			unit := testutils.ParseString(t, test.unit)
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
	t.Parallel()

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

			unit := testutils.ParseString(t, test.unit)
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
	t.Parallel()

	vRef := testutils.NewTestValidator(V.Options{CheckReferences: true}, "test.network", "test.container")
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
		{"BadReference", "[Container]\nNetwork=bad.container", vRef, []V.Location{{Line: 2, Column: 8}}},
		{"BadReferences", "[Container]\nNetwork=bad.container\nOther=6\nNetwork=otherbad.network", vRef,
			[]V.Location{{Line: 2, Column: 8}, {Line: 4, Column: 8}}},
	}

	rule := CanReference(M.UnitTypeNetwork, M.UnitTypeContainer)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := testutils.ParseString(t, test.unit)
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
	t.Parallel()

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
		{"FieldBadFormat", "[Container]\nNetwork=test:", []V.Location{{Line: 2, Column: 8}}},
		{"FieldBadFormat2", "[Container]\nNetwork=test:opt1,", []V.Location{{Line: 2, Column: 8}}},
		{"FieldBadFormat3", "[Container]\nNetwork=test:test2:test3", []V.Location{{Line: 2, Column: 8}}},
	}

	format := Format{Name: "TestFormat", ValueSeparator: ":", OptionsSeparator: ","}
	rule := HaveFormat(format)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := testutils.ParseString(t, test.unit)
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
	t.Parallel()

	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldHasSomeAllowedValues", "[Container]\nPublishPort=val1\nPublishPort=val2", nil},
		{"FieldHasAllAllowedValues", "[Container]\nPublishPort=val1\nPublishPort=val2\nPublishPort=val3", nil},
		{"FieldHasBadValues", "[Container]\nPublishPort=bad\nPublishPort=bad2",
			[]V.Location{{Line: 2, Column: 12}, {Line: 3, Column: 12}}},
		{"FieldHasSomeBadValues", "[Container]\nPublishPort=bad\nOther=test\nPublishPort=val2\nPublishPort=bad2",
			[]V.Location{{Line: 2, Column: 12}, {Line: 5, Column: 12}}},
	}

	rule := AllowedValues("val1", "val2", "val3")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := testutils.ParseString(t, test.unit)
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
	t.Parallel()

	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldWithSuffix", "[Container]\nPod=test.pod", nil},
		{"FieldWithoutSuffix", "[Container]\nPod=bad", []V.Location{{Line: 2, Column: 4}}},
	}

	rule := HasSuffix(".pod")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := testutils.ParseString(t, test.unit)
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
	t.Parallel()

	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldWithDependency", "[Container]\nUser=1\nGroup=1", nil},
		{"FieldWithoutDependency", "[Container]\nGroup=1", []V.Location{{Line: 2, Column: 0}}},
	}

	rule := DependsOn(container.User)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := testutils.ParseString(t, test.unit)
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
	t.Parallel()

	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"DeprecatedFieldPresent", "[Container]\nRemapUid=1\nOther=1\nRemapUid=2",
			[]V.Location{{Line: 2, Column: 0}, {Line: 4, Column: 0}}},
	}

	rule := Deprecated
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := testutils.ParseString(t, test.unit)
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
	t.Parallel()

	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldMatchesRegexp", "[Container]\nRemapUid=1\nOther=1\nRemapUid=2", nil},
		{"FieldDoesNotMatchRegexp", "[Container]\nRemapUid=1\nOther=1\nRemapUid=abcd", []V.Location{{Line: 4, Column: 9}}},
	}

	rule := MatchRegexp(regexp.MustCompile(`\d+`))
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := testutils.ParseString(t, test.unit)
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
	t.Parallel()

	tests := []struct {
		name   string
		unit   string
		errors []V.Location
	}{
		{"NoErrorsIfFieldAbsent", "[Container]\nOther=test", nil},
		{"FieldMatchesCondition", "[Container]\nRemapUid=1\nUser=test", nil},
		{"FieldMatchesCondition2", "[Container]\nUser=test", nil},
		{"NoErrorsIfRulePredicateIsFalse", "[Container]\nRemapUid=1\nRemapUid=2\nUser=other", nil},
		{"ConditionNotRespected", "[Container]\nRemapUid=1\nRemapUid=2\nUser=test", []V.Location{{Line: 2, Column: 9}}},
	}

	rule := ValuesMust(HaveZeroOrOneValues, WhenFieldEquals(container.User, "test"),
		"field %s should have zero or one value when field %s have value 'test'", container.RemapUid.Key, container.User.Key)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := testutils.ParseString(t, test.unit)
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
	t.Parallel()

	tests := []struct {
		name  string
		unit  string
		error *V.Location
	}{
		{"HasOneValue", "[Container]\nRemapUid=1", nil},
		{"HasZeroValues", "[Container]\nOther=test", nil},
		{"HasManyValues", "[Container]\nRemapUid=1\nRemapUid=5", &V.Location{Line: 3, Column: 9}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			unit := testutils.ParseString(t, test.unit)
			res, _ := unit.Lookup(container.RemapUid)

			if err := HaveZeroOrOneValues(v, container.RemapUid, res.Values()); err != nil {
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
	t.Parallel()

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

			unit := testutils.ParseString(t, test.unit)
			result := rule(v, unit, container.User)
			assert.Equal(t, test.result, result)
		})
	}
}
