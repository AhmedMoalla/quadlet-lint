package quadlet

import (
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

const ValidatorName = "quadlet"

var (
	AmbiguousImageName = validator.NewErrorType("ambiguous-image-name", validator.LevelWarning, ValidatorName)
	UnknownKey         = validator.NewErrorType("unknown-key", validator.LevelError, ValidatorName)
	RequiredKey        = validator.NewErrorType("required-key", validator.LevelError, ValidatorName)
	KeyConflict        = validator.NewErrorType("key-conflict", validator.LevelError, ValidatorName)
	InvalidValue       = validator.NewErrorType("invalid-value", validator.LevelError, ValidatorName)
	DeprecatedKey      = validator.NewErrorType("deprecated-key", validator.LevelError, ValidatorName)
)

var validators = map[parser.UnitType]validator.Validator{
	parser.UnitTypeContainer: ContainerValidator{},
	parser.UnitTypeVolume:    noOpValidator{},
	parser.UnitTypeKube:      noOpValidator{},
	parser.UnitTypeNetwork:   noOpValidator{},
	parser.UnitTypeImage:     noOpValidator{},
	parser.UnitTypeBuild:     noOpValidator{},
	parser.UnitTypePod:       noOpValidator{},
}

type Validator struct{}

func (q Validator) Validate(unit parser.UnitFile) []validator.ValidationError {
	return validators[unit.UnitType].Validate(unit)
}

type noOpValidator struct{}

func (n noOpValidator) Validate(unit parser.UnitFile) []validator.ValidationError {
	return []validator.ValidationError{}
}
