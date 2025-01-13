package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitValueAppend(t *testing.T) {
	t.Parallel()

	values := make([]unitValue, 0)
	value := unitValue{
		key:         "TestKey",
		value:       "TestValue     TestValue2    TestValue3",
		line:        10,
		valueColumn: 5,
	}
	values, err := splitValueAppend(values, value, WhitespaceSeparators, SplitRelax)
	if err != nil {
		t.Errorf("Error happened while calling splitValueAppend: %s", err)
	}

	expectedValues := []unitValue{
		{key: "TestKey", value: "TestValue", line: 10, valueColumn: 5},
		{key: "TestKey", value: "TestValue2", line: 10, valueColumn: 19},
		{key: "TestKey", value: "TestValue3", line: 10, valueColumn: 33},
	}

	assert.Equal(t, expectedValues, values)
}
