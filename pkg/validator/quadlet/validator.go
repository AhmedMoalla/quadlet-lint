package quadlet

import (
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

const ValidatorName = "quadlet"

var (
	AmbiguousImageName = validator.NewErrorType("ambiguous-image-name", validator.LevelWarning)
)

func Validator(units []parser.UnitFile, options validator.Options) validator.Validator {
	context := validator.Context{
		AllUnitFiles: units,
		Options:      options,
	}
	return quadletValidator{
		name:    ValidatorName,
		context: context,
		validators: map[parser.UnitType]validator.Validator{
			parser.UnitTypeContainer: containerValidator{name: "container", context: context},
			parser.UnitTypeVolume:    noOpValidator{},
			parser.UnitTypeKube:      noOpValidator{},
			parser.UnitTypeNetwork:   noOpValidator{},
			parser.UnitTypeImage:     noOpValidator{},
			parser.UnitTypeBuild:     noOpValidator{},
			parser.UnitTypePod:       noOpValidator{},
		},
	}
}

type quadletValidator struct {
	name       string
	context    validator.Context
	validators map[parser.UnitType]validator.Validator
}

func (v quadletValidator) Name() string {
	return v.name
}

func (v quadletValidator) Context() validator.Context {
	return v.context
}

func (v quadletValidator) Validate(unit parser.UnitFile) []validator.ValidationError {
	return v.validators[unit.UnitType].Validate(unit)
}

type noOpValidator struct{}

func (v noOpValidator) Name() string {
	return "noop"
}

func (v noOpValidator) Context() validator.Context {
	return validator.Context{}
}

func (v noOpValidator) Validate(parser.UnitFile) []validator.ValidationError {
	return []validator.ValidationError{}
}
