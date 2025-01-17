package parser

import (
	"maps"
	"os"
	"testing"

	. "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	generated "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/container"
	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/lookup"
	"github.com/AhmedMoalla/quadlet-lint/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUnitFileErrors(t *testing.T) {
	t.Parallel()

	file, errors := ParseUnitFile("not found")
	assert.Nil(t, file)
	assert.Len(t, errors, 1)
	require.ErrorIs(t, errors[0].inner, os.ErrNotExist)

	file, errors = ParseUnitFile("testdata/err.container")
	assert.Nil(t, file)

	expectedErrs := []ParsingError{
		{Group: "", Key: "", Line: 1, Column: 0},
		{Group: "Container", Key: "ContainerName", Line: 5, Column: 14},
		{Group: "Service", Key: "Bla", Line: 14, Column: 4},
		{Group: "Install", Key: "afaffazf", Line: 20, Column: 0},
		{Group: "", Key: "", Line: 22, Column: 1},
		{Group: "Group", Key: "=value", Line: 26, Column: 0},
	}
	assert.Len(t, errors, len(expectedErrs), "expected %d errors got %d errors",
		len(expectedErrs), len(errors))

	for i, err := range errors {
		expected := expectedErrs[i]
		assert.Equal(t, expected.Group, err.Group)
		assert.Equal(t, expected.Key, err.Key)
		assert.Equal(t, expected.Line, err.Line)
		assert.Equal(t, expected.Column, err.Column)
	}
}

func TestParseUnitFile(t *testing.T) {
	t.Parallel()

	file, errors := ParseUnitFile("testdata/unit.container")
	assertUnitFileParsedCorrectly(t, file, errors)
}

func TestParseUnitFileString(t *testing.T) {
	t.Parallel()

	bytes, err := os.ReadFile("testdata/unit.container")
	require.NoError(t, err)
	file, errors := ParseUnitFileString("unit.container", string(bytes))
	assert.Equal(t, "unit.container", file.FileName())
	assert.Equal(t, UnitTypeContainer, file.UnitType())
	assertUnitFileParsedCorrectly(t, file, errors)
}

var expectedGroups = map[string]map[string][]UnitValue{
	"Container": {
		container.Image.Key:       {UnitValue{Key: container.Image.Key, Value: "my-image", Line: 3, Column: 6}},
		container.PublishPort.Key: {UnitValue{Key: container.PublishPort.Key, Value: "8080:8080/tcp", Line: 4, Column: 12}},
		container.Network.Key:     {UnitValue{Key: container.Network.Key, Value: "my-network", Line: 6, Column: 15}},
		container.Environment.Key: {
			UnitValue{Key: "env1", Value: "value1", Line: 8, Column: 12},
			UnitValue{Key: "env2", Value: "value2", Line: 8, Column: 12}},
		container.ReadOnly.Key: {UnitValue{Key: container.ReadOnly.Key, Value: "true", Line: 12, Column: 9}},
		container.EnvironmentFile.Key: {
			UnitValue{Key: container.EnvironmentFile.Key, Value: "env1", Line: 14, Column: 16},
			UnitValue{Key: container.EnvironmentFile.Key, Value: "env2", Line: 14, Column: 21},
			UnitValue{Key: container.EnvironmentFile.Key, Value: "env3", Line: 15, Column: 16},
		},
		container.Exec.Key: {UnitValue{Key: container.Exec.Key, Value: "value", Line: 19, Column: 5}},
	},
	"Service": {
		"Restart":         {UnitValue{Key: "Restart", Value: "always", Line: 23, Column: 8}},
		"TimeoutStartSec": {UnitValue{Key: "TimeoutStartSec", Value: "900", Line: 25, Column: 16}},
	},
	"Install": {
		"WantedBy": {
			UnitValue{Key: "WantedBy", Value: "multi-user.target", Line: 29, Column: 9},
			UnitValue{Key: "WantedBy", Value: "default.target", Line: 29, Column: 27}},
	},
}

var additionalFields = map[string]map[string]Field{
	"Service": {
		"Restart":         Field{Group: "Service", Key: "Restart", LookupFunc: lookup.Lookup},
		"TimeoutStartSec": Field{Group: "Service", Key: "TimeoutStartSec", LookupFunc: lookup.LookupUint32},
	},
	"Install": {
		"WantedBy": Field{Group: "Install", Key: "WantedBy", LookupFunc: lookup.LookupAllStrv},
	},
}

func assertUnitFileParsedCorrectly(t *testing.T, file UnitFile, errors []ParsingError) {
	t.Helper()

	if file == nil {
		t.Fatal("parsed file is nil")
	}
	assert.Empty(t, errors)

	allFields := make(FieldsMap, len(generated.Fields))
	maps.Copy(allFields, generated.Fields)
	allFields = utils.MergeMaps(allFields, additionalFields)

	for group, kv := range expectedGroups {
		for key, expectedValues := range kv {
			field := allFields[group][key]
			result, ok := file.Lookup(field)
			assert.True(t, ok, "expected key '%s' to have values '%s' but lookup was not successful",
				key, expectedValues)
			assert.ElementsMatch(t, expectedValues, result.Values())
		}
	}

	result, ok := file.Lookup(container.ReadOnly)
	assert.True(t, ok)
	assert.True(t, result.BoolValue())

	result, ok = file.Lookup(additionalFields["Service"]["TimeoutStartSec"])
	assert.True(t, ok)
	assert.Equal(t, 900, result.IntValue())
}

func TestKeyNameIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key    string
		valid  bool
		errPos int
	}{
		{"test", true, -1},
		{"", false, 0},
		{"key=val", false, 3},
		{"  test", false, 0},
		{"test   ", false, 6},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			t.Parallel()

			valid, errPos := keyNameIsValid(test.key)
			assert.Equal(t, test.valid, valid)
			assert.Equal(t, test.errPos, errPos)
		})
	}
}

func TestGroupNameIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key    string
		valid  bool
		errPos int
	}{
		{"test", true, -1},
		{"[Group]", false, 0},
		{"", false, 0},
		{"[G", false, 0},
		{"G]", false, 1},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			t.Parallel()

			valid, errPos := groupNameIsValid(test.key)
			assert.Equal(t, test.valid, valid)
			assert.Equal(t, test.errPos, errPos)
		})
	}
}

func TestLineIsGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key   string
		valid bool
	}{
		{"[Group]", true},
		{"[Group] test", false},
		{"test", false},
		{"", false},
		{"[G", false},
		{"G]", false},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			t.Parallel()

			valid := lineIsGroup(test.key)
			assert.Equal(t, test.valid, valid)
		})
	}
}
