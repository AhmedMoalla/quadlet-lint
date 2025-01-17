package common

import (
	"fmt"

	M "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	generated "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

func ValidatorWithFields(fields M.FieldsMap) V.Validator {
	return commonValidator{
		context: V.Context{
			AllFields: fields,
		},
	}
}

func Validator() V.Validator {
	return ValidatorWithFields(generated.Fields)
}

type commonValidator struct {
	context V.Context
}

func (v commonValidator) Name() string {
	return "common"
}

func (v commonValidator) Context() V.Context {
	return v.context
}

var ignoredSystemdGroups = map[string]bool{"Service": true, "Install": true, "Unit": true}

func (v commonValidator) Validate(unit M.UnitFile) []V.ValidationError {
	validationErrors := make([]V.ValidationError, 0)
	for _, group := range unit.ListGroups() {
		if _, ignored := ignoredSystemdGroups[group]; ignored {
			continue
		}

		allowedFields := v.Context().AllFields[group]
		for _, key := range unit.ListKeys(group) {
			if _, ok := allowedFields[key.Key]; !ok {
				validationErrors = append(validationErrors, *V.UnknownKey.Err(v.Name(), group, key.Key, key.Line, 0,
					fmt.Sprintf("key '%s' is not allowed in group '%s'", key.Key, group)))
			}
		}
	}
	return validationErrors
}
