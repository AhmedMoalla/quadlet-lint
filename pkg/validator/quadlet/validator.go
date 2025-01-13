package quadlet

import (
	"github.com/AhmedMoalla/quadlet-lint/pkg/model"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

const ValidatorName = "quadlet"

var (
	AmbiguousImageName = validator.NewErrorType("ambiguous-image-name", validator.LevelWarning)
)

func Validator(units []model.UnitFile, options validator.Options) validator.Validator {
	context := validator.Context{
		AllUnitFiles: units,
		Options:      options,
	}
	return quadletValidator{
		name:    ValidatorName,
		context: context,
		validators: map[model.UnitType]validator.Validator{
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

type quadletValidator struct {
	name       string
	context    validator.Context
	validators map[model.UnitType]validator.Validator
}

func (v quadletValidator) Name() string {
	return v.name
}

func (v quadletValidator) Context() validator.Context {
	return v.context
}

func (v quadletValidator) Validate(unit model.UnitFile) []validator.ValidationError {
	return v.validators[unit.UnitType()].Validate(unit)
}

type noOpValidator struct{}

func (v noOpValidator) Name() string {
	return "noop"
}

func (v noOpValidator) Context() validator.Context {
	return validator.Context{}
}

func (v noOpValidator) Validate(model.UnitFile) []validator.ValidationError {
	return []validator.ValidationError{}
}
