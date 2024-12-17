package quadlet

import (
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

type ContainerValidator struct{}

func (c ContainerValidator) Validate(unit parser.UnitFile) []validator.ValidationError {
	validationErrors := make([]validator.ValidationError, 0)

	validationErrors = appendError(validationErrors, warnIfAmbiguousName(unit, ContainerGroup))
	validationErrors = appendError(validationErrors, checkForUnknownKeys(&unit, ContainerGroup, supportedContainerKeys))
	validationErrors = appendError(validationErrors, checkForMissingImage(unit))

	return validationErrors
}

func checkForMissingImage(unit parser.UnitFile) *validator.ValidationError {
	// One image or rootfs must be specified for the container
	image, _ := unit.Lookup(ContainerGroup, KeyImage)
	rootfs, _ := unit.Lookup(ContainerGroup, KeyRootfs)
	if len(image) == 0 && len(rootfs) == 0 {
		return validator.Error(RequiredKey, 0, 0, "no Image or Rootfs key specified")
	}
	if len(image) > 0 && len(rootfs) > 0 {
		return validator.Error(KeyConflict, 0, 0,
			"the Image And Rootfs keys conflict can not be specified together")
	}

	return nil
}
