package testutils

import (
	"testing"

	P "github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

func ParseString(t *testing.T, content string) P.UnitFile {
	t.Helper()
	unit, errs := P.ParseUnitFileString("test.container", content)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Error(err.Error())
		}
		t.Fatal("errors while parsing file content")
	}

	return *unit
}

type testValidator struct {
	ctx V.Context
}

func (t testValidator) Name() string {
	return "test"
}

func (t testValidator) Context() V.Context {
	return t.ctx
}

func (t testValidator) Validate(_ P.UnitFile) []V.ValidationError {
	return nil
}

func NewTestValidator(options V.Options, files ...string) V.Validator {
	units := make([]P.UnitFile, 0, len(files))
	for _, file := range files {
		units = append(units, P.UnitFile{Filename: file})
	}
	return testValidator{ctx: V.Context{
		Options:      options,
		AllUnitFiles: units,
	}}
}
