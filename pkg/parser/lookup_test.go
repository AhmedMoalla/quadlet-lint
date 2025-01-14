package parser

import (
	"testing"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupLastArgs(t *testing.T) {
	t.Parallel()

	content := `[Container]
Exec=first
Exec=/some/path "an arg" "a;b\nc\td'e" a;b\nc\td 'a"b' '\110ello \127orld'`

	unit, err := ParseUnitFileString("test.container", content)
	require.Empty(t, err)

	expected := []string{"/some/path", "an arg", "a;b\nc\td'e", "a;b\nc\td", "a\"b", "Hello World"}
	res, ok := unit.Lookup(container.Exec)
	assert.True(t, ok)
	assert.Len(t, res.Values(), len(expected))
	for i := range expected {
		assert.Equal(t, expected[i], res.Values()[i].Value)
	}
}
