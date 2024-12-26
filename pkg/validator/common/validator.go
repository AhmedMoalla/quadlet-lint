package common

import (
	"fmt"

	model "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

func Validator() V.Validator {
	return commonValidator{}
}

type commonValidator struct{}

func (v commonValidator) Name() string {
	return "common"
}

func (v commonValidator) Context() V.Context {
	return V.Context{}
}

var ignoreGroups = map[string]bool{"Service": true, "Install": true, "Unit": true}

// TODO: line and column position not implemented
func (v commonValidator) Validate(unit parser.UnitFile) []V.ValidationError {
	validationErrors := make([]V.ValidationError, 0)
	for _, group := range unit.ListGroups() {
		if _, ignored := ignoreGroups[group]; ignored {
			continue
		}

		allowedFields := model.Fields[group]
		for _, key := range unit.ListKeys(group) {
			if _, ok := allowedFields[key]; !ok {
				validationErrors = append(validationErrors, *V.Err(v.Name(), V.UnknownKey, 0, 0,
					fmt.Sprintf("key '%s' is not allowed in group '%s'", key, group)))
			}

			if res, ok := unit.Lookup(allowedFields[key]); ok && len(res.Values) == 0 {
				validationErrors = append(validationErrors, *V.Err(v.Name(), V.EmptyValue, 0, 0,
					fmt.Sprintf("key '%s' in group '%s' has an empty value", key, group)))
			}

		}
	}
	return validationErrors
}
