package quadlet

import (
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

const ValidatorName = "quadlet"

var (
	AmbiguousImageName = validator.NewErrorType("ambiguous-image-name", validator.LevelWarning)
)

var validators = map[parser.UnitType]validator.Validator{
	parser.UnitTypeContainer: containerValidator{ValidatorName: "container"},
	parser.UnitTypeVolume:    noOpValidator{},
	parser.UnitTypeKube:      noOpValidator{},
	parser.UnitTypeNetwork:   noOpValidator{},
	parser.UnitTypeImage:     noOpValidator{},
	parser.UnitTypeBuild:     noOpValidator{},
	parser.UnitTypePod:       noOpValidator{},
}

func Validator() validator.Validator {
	return quadletValidator{ValidatorName: ValidatorName}
}

type quadletValidator struct {
	ValidatorName string
}

func (q quadletValidator) Validate(unit parser.UnitFile) []validator.ValidationError {
	return validators[unit.UnitType].Validate(unit)
}

type noOpValidator struct{}

func (n noOpValidator) Validate(unit parser.UnitFile) []validator.ValidationError {
	return []validator.ValidationError{}
}
