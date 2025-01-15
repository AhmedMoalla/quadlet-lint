package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeMaps(t *testing.T) {
	map1 := map[string]string{
		"a": "b",
		"c": "d",
	}
	map2 := map[string]string{
		"a": "b",
		"e": "f",
		"g": "h",
	}
	map3 := MergeMaps(map1, map2)
	assert.Len(t, map3, 4)
	assert.Equal(t, "b", map3["a"])
	assert.Equal(t, "d", map3["c"])
	assert.Equal(t, "f", map3["e"])
	assert.Equal(t, "h", map3["g"])
}

func TestReverseMap(t *testing.T) {
	src := map[string]string{
		"a": "b",
		"c": "d",
	}
	dst := ReverseMap(src)
	assert.Len(t, dst, len(src))
	assert.Equal(t, "a", dst["b"])
	assert.Equal(t, "c", dst["d"])
}
