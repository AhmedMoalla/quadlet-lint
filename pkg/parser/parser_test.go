package parser

import (
	"os"
	"testing"

	. "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	generated "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/lookup"
	"github.com/AhmedMoalla/quadlet-lint/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestParseUnitFileErrors(t *testing.T) {
	t.Parallel()

	file, errors := ParseUnitFile("not found")
	assert.Nil(t, file)
	assert.Len(t, errors, 1)
	assert.ErrorIs(t, errors[0].inner, os.ErrNotExist)

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

	file, errors := ParseUnitFile("testdata/httpbin.container")
	assertUnitFileParsedCorrectly(t, file, errors)
}

func TestParseUnitFileString(t *testing.T) {
	t.Parallel()

	bytes, err := os.ReadFile("testdata/httpbin.container")
	assert.Nil(t, err)
	file, errors := ParseUnitFileString("httpbin.container", string(bytes))
	assertUnitFileParsedCorrectly(t, file, errors)
}

func assertUnitFileParsedCorrectly(t *testing.T, file UnitFile, errors []ParsingError) {
	t.Helper()

	if file == nil {
		t.Fatal("parsed file is nil")
	}
	assert.Empty(t, errors)

	expectedGroups := map[string]map[string][]UnitValue{
		"Container": {
			"Image":       {UnitValue{Key: "Image", Value: "docker.io/kennethreitz/httpbin", Line: 3, Column: 6}},
			"PublishPort": {UnitValue{Key: "PublishPort", Value: "8080:8080/tcp", Line: 4, Column: 12}},
			"Network":     {UnitValue{Key: "Network", Value: "my-network", Line: 6, Column: 15}},
			"Environment": {
				UnitValue{Key: "env1", Value: "value1", Line: 7, Column: 12},
				UnitValue{Key: "env2", Value: "value2", Line: 7, Column: 12}},
		},
		"Service": {
			"Restart":         {UnitValue{Key: "Restart", Value: "always", Line: 12, Column: 8}},
			"TimeoutStartSec": {UnitValue{Key: "TimeoutStartSec", Value: "900", Line: 13, Column: 16}},
		},
		"Install": {
			"WantedBy": {
				UnitValue{Key: "WantedBy", Value: "multi-user.target", Line: 17, Column: 9},
				UnitValue{Key: "WantedBy", Value: "default.target", Line: 17, Column: 27}},
		},
	}

	additionalFields := map[string]map[string]Field{
		"Service": {
			"Restart":         Field{Group: "Service", Key: "Restart", LookupFunc: lookup.Lookup},
			"TimeoutStartSec": Field{Group: "Service", Key: "TimeoutStartSec", LookupFunc: lookup.LookupUint32},
		},
		"Install": {
			"WantedBy": Field{Group: "Install", Key: "WantedBy", LookupFunc: lookup.LookupAllStrv},
		},
	}
	generated.Fields = utils.MergeMaps(generated.Fields, additionalFields)

	for group, kv := range expectedGroups {
		for key, expectedValues := range kv {
			field := generated.Fields[group][key]
			result, ok := file.Lookup(field)
			assert.True(t, ok, "expected key '%s' to have values '%s' but lookup was not successful",
				key, expectedValues)
			assert.ElementsMatch(t, expectedValues, result.Values())
		}
	}
}

func TestKeyNameIsValid(t *testing.T) {
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
