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

func TestCUnescapeOne(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		in        string
		acceptNul bool

		ret      rune
		count    int
		eightBit bool
	}{
		{name: "empty", in: "", ret: 0, count: -1},
		{name: `invalid \k`, in: "k", ret: 0, count: -1},
		{name: `\a`, in: "a", ret: '\a', count: 1},
		{name: `\b`, in: "b", ret: '\b', count: 1},
		{name: `\f`, in: "f", ret: '\f', count: 1},
		{name: `\n`, in: "n", ret: '\n', count: 1},
		{name: `\r`, in: "r", ret: '\r', count: 1},
		{name: `\t`, in: "t", ret: '\t', count: 1},
		{name: `\v`, in: "v", ret: '\v', count: 1},
		{name: `\\`, in: "\\", ret: '\\', count: 1},
		{name: `"`, in: "\"", ret: '"', count: 1},
		{name: `'`, in: "'", ret: '\'', count: 1},
		{name: `\s`, in: "s", ret: ' ', count: 1},
		{name: `too short \x1`, in: "x1", ret: 0, count: -1},
		{name: `invalid hex \xzz`, in: "xzz", ret: 0, count: -1},
		{name: `invalid hex \xaz`, in: "xaz", ret: 0, count: -1},
		{name: `\xAb1`, in: "xAb1", ret: '«', count: 3, eightBit: true},
		{name: `\x000 acceptNul=false`, in: "x000", ret: 0, count: -1},
		{name: `\x000 acceptNul=true`, in: "x000", ret: 0, count: 3, eightBit: true, acceptNul: true},
		{name: `too short \u123`, in: "u123", ret: 0, count: -1},
		{name: `\u2a00`, in: "u2a00", ret: '⨀', count: 5},
		{name: `invalid hex \u12v1A`, in: "u12v1A", ret: 0, count: -1},
		{name: `\u0000 acceptNul=false`, in: "u0000", ret: 0, count: -1},
		{name: `\u0000 acceptNul=true`, in: "u0000", ret: 0, count: 5, acceptNul: true},
		{name: `too short \U123`, in: "U123", ret: 0, count: -1},
		{name: `invalid unicode \U12345678`, in: "U12345678", ret: 0, count: -1},
		{name: `invalid hex \U1234V678`, in: "U1234V678", ret: 0, count: -10},
		{name: `\U0001F51F`, in: "U0001F51F", ret: '🔟', count: 9},
		{name: `\U00000000 acceptNul=false`, in: "U00000000", ret: 0, count: -1, acceptNul: false},
		{name: `\U00000000 acceptNul=true`, in: "U00000000", ret: 0, count: 9, acceptNul: true},
		{name: `too short 77`, in: "77", ret: 0, count: -1},
		{name: `invalid octal 792`, in: "792", ret: 0, count: -1},
		{name: `invalid octal 758`, in: "758", ret: 0, count: -1},
		{name: `000 acceptNul=false`, in: "000", ret: 0, count: -1},
		{name: `000 acceptNul=true`, in: "000", ret: 0, count: 3, acceptNul: true, eightBit: true},
		{name: `too big 777 > 255 bytes`, in: "777", ret: 0, count: -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			count, out, eightBit := cUnescapeOne(test.in, test.acceptNul)
			assert.Equal(t, test.count, count)
			assert.Equal(t, test.ret, out)
			assert.Equal(t, test.eightBit, eightBit)
		})
	}
}
