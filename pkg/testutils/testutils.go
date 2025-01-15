package testutils

import (
	"testing"

	M "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	P "github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/utils"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

func ParseString(t *testing.T, content string) M.UnitFile {
	t.Helper()
	unit, errs := P.ParseUnitFileString("test.container", content)
	if len(errs) != 0 {
		for _, err := range errs {
			t.Error(err.Error())
		}
		t.Fatal("errors while parsing file content")
	}

	return unit
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

func (t testValidator) Validate(_ M.UnitFile) []V.ValidationError {
	return nil
}

func NewTestValidator(options V.Options, files ...string) V.Validator {
	units := make([]M.UnitFile, 0, len(files))
	for _, file := range files {
		units = append(units, testUnitFile{filename: file})
	}
	return testValidator{ctx: V.Context{
		Options:      options,
		AllUnitFiles: units,
	}}
}

var IncludedTestUnits = generateFilePerExtension()

func generateFilePerExtension() []M.UnitFile {
	return utils.MapSlice(M.AllUnitFileExtensions, func(ext string) M.UnitFile {
		return testUnitFile{filename: "test" + ext}
	})
}

type testUnitFile struct {
	filename string
}

func (t testUnitFile) FileName() string {
	return t.filename
}

func (t testUnitFile) UnitType() M.UnitType {
	panic("implement me")
}

func (t testUnitFile) Lookup(field M.Field) (M.LookupResult, bool) {
	panic("implement me")
}

func (t testUnitFile) HasGroup(groupName string) bool {
	panic("implement me")
}

func (t testUnitFile) ListGroups() []string {
	panic("implement me")
}

func (t testUnitFile) ListKeys(groupName string) []M.UnitKey {
	panic("implement me")
}

func (t testUnitFile) HasKey(field M.Field) bool {
	panic("implement me")
}

func (t testUnitFile) HasValue(field M.Field) bool {
	panic("implement me")
}
