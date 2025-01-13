package parser

import (
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model"
	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnitFile_HasGroup(t *testing.T) {
	t.Parallel()

	unit, errors := ParseUnitFile("testdata/httpbin.container")
	require.Empty(t, errors)
	assert.True(t, unit.HasGroup("Container"))
	assert.False(t, unit.HasGroup("Pod"))
}

func TestUnitFile_ListGroups(t *testing.T) {
	t.Parallel()

	unit, errors := ParseUnitFile("testdata/httpbin.container")
	require.Empty(t, errors)
	assert.ElementsMatch(t, []string{"Container", "Service", "Install"}, unit.ListGroups())
}

func TestUnitFile_ListKeys(t *testing.T) {
	t.Parallel()

	unit, errors := ParseUnitFile("testdata/httpbin.container")
	require.Empty(t, errors)

	expectedKeys := []model.UnitKey{
		{Key: "ContainerName", Line: 2},
		{Key: "Image", Line: 3},
		{Key: "PublishPort", Line: 4},
		{Key: "Network", Line: 6},
		{Key: "Environment", Line: 7},
	}
	assert.ElementsMatch(t, expectedKeys, unit.ListKeys("Container"))
}

func TestUnitFile_HasValue(t *testing.T) {
	t.Parallel()

	unit, errors := ParseUnitFileString("test.container", "[Container]\nContainerName=name")
	require.Empty(t, errors)
	assert.True(t, unit.HasValue(container.ContainerName))
	assert.False(t, unit.HasValue(container.Pod))
}

func TestUnitFile_HasKey(t *testing.T) {
	t.Parallel()

	unit, errors := ParseUnitFileString("test.container", "[Container]\nContainerName=name")
	require.Empty(t, errors)
	assert.True(t, unit.HasKey(container.ContainerName))
	assert.False(t, unit.HasValue(container.Pod))
}
