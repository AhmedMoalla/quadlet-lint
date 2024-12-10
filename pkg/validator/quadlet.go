package validator

import (
	"github.com/containers/podman/v5/pkg/systemd/parser"
)

type QuadletValidator struct {
}

func (q QuadletValidator) Validate(unitFile parser.UnitFile) []ValidationError {

	return nil
}
