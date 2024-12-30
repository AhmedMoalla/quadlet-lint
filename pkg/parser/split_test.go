package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitValueAppend(t *testing.T) {
	t.Parallel()

	values := make([]UnitValue, 0)
	value := UnitValue{
		Key:    "TestKey",
		Value:  "TestValue     TestValue2    TestValue3",
		Line:   10,
		Column: 5,
	}
	values, err := splitValueAppend(values, value, WhitespaceSeparators, SplitRelax)
	if err != nil {
		t.Errorf("Error happened while calling splitValueAppend: %s", err)
	}

	expectedValues := []UnitValue{
		{Key: "TestKey", Value: "TestValue", Line: 10, Column: 5},
		{Key: "TestKey", Value: "TestValue2", Line: 10, Column: 19},
		{Key: "TestKey", Value: "TestValue3", Line: 10, Column: 33},
	}

	assert.Equal(t, len(expectedValues), len(values))

	for i, expected := range expectedValues {
		assert.Equal(t, expected, values[i])
	}
}
