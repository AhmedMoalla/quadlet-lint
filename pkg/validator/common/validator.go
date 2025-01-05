package common

import (
	"fmt"
	"log/slog"

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

var ignoredGroups = map[string]bool{"Service": true, "Install": true, "Unit": true}

func (v commonValidator) Validate(unit parser.UnitFile) []V.ValidationError {
	logger := slog.With("validator", v.Name(), "unitFile", unit.FilePath)
	logger.Debug("validating file")
	validationErrors := make([]V.ValidationError, 0)
	for _, group := range unit.ListGroups() {
		if ignoredGroups[group] {
			logger.Debug("ignoring group", "group", group)
			continue
		}

		logger.Debug("checking if all parsed fields are allowed in group", "group", group)
		allowedFields := model.Fields[group]
		for _, key := range unit.ListKeys(group) {
			if _, ok := allowedFields[key.Key]; !ok {
				msg := fmt.Sprintf("key '%s' is not allowed in group '%s'", key.Key, group)
				validationErrors = append(validationErrors, *V.Err(v.Name(), V.UnknownKey, key.Line, 0, msg))
				logger.Debug(msg)
				continue
			}

			if res, ok := unit.Lookup(allowedFields[key.Key]); ok && len(res.Values) == 0 {
				msg := fmt.Sprintf("key '%s' in group '%s' has an empty value", key.Key, group)
				validationErrors = append(validationErrors, *V.Err(v.Name(), V.EmptyValue, key.Line, 0, msg))
				logger.Debug(msg)
			}
		}
	}
	return validationErrors
}
