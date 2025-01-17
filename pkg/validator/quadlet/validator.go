package quadlet

import (
	"github.com/AhmedMoalla/quadlet-lint/pkg/model"
	generated "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

const ValidatorName = "quadlet"

var (
	AmbiguousImageName = V.NewErrorCategory("ambiguous-image-name", V.LevelWarning)
)

func ValidatorWithFields(fields model.FieldsMap, units []model.UnitFile, options V.Options) V.Validator {
	context := V.Context{
		AllFields:    fields,
		AllUnitFiles: units,
		Options:      options,
	}
	return quadletValidator{
		name:    ValidatorName,
		context: context,
		validators: map[model.UnitType]V.Validator{
			model.UnitTypeContainer: containerValidator{name: "container", context: context},
			model.UnitTypeVolume:    noOpValidator{},
			model.UnitTypeKube:      noOpValidator{},
			model.UnitTypeNetwork:   noOpValidator{},
			model.UnitTypeImage:     noOpValidator{},
			model.UnitTypeBuild:     noOpValidator{},
			model.UnitTypePod:       noOpValidator{},
		},
	}
}

func Validator(units []model.UnitFile, options V.Options) V.Validator {
	return ValidatorWithFields(generated.Fields, units, options)
}

type quadletValidator struct {
	name       string
	context    V.Context
	validators map[model.UnitType]V.Validator
}

func (v quadletValidator) Name() string {
	return v.name
}

func (v quadletValidator) Context() V.Context {
	return v.context
}

func (v quadletValidator) Validate(unit model.UnitFile) []V.ValidationError {
	return v.validators[unit.UnitType()].Validate(unit)
}

type noOpValidator struct{}

func (v noOpValidator) Name() string {
	return "noop"
}

func (v noOpValidator) Context() V.Context {
	return V.Context{}
}

func (v noOpValidator) Validate(model.UnitFile) []V.ValidationError {
	return []V.ValidationError{}
}
