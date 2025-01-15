package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapSlice(t *testing.T) {
	type test struct {
		prop1 string
	}

	src := []test{
		{prop1: "test"},
		{prop1: "test1"},
		{prop1: "test2"},
	}

	dst := MapSlice(src, func(test test) string {
		return test.prop1
	})
	expected := []string{"test", "test1", "test2"}
	require.Len(t, dst, len(expected))
	for i := range expected {
		assert.Equal(t, expected[i], dst[i])
	}
}
