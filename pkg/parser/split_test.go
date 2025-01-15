package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{name: "376", in: "376", ret: 'þ', count: 3, eightBit: true},
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

//nolint:funlen
func TestExtractFirstWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		expected      []string
		flags         SplitFlags
		expectedError error
		separators    string
	}{
		{
			name: "SplitRelax", flags: SplitRelax,
			input:    `"unbalanced quotes   \`,
			expected: []string{"\"unbalanced", "quotes"},
		},
		{
			name:          "No flags trailing backslash",
			input:         `unbalanced quotes \`,
			expectedError: errUnbalancedEscape,
		},
		{
			name: "SplitCUnescape", flags: SplitCUnescape,
			input: `\a \b \f \n \r \t \v \\ \" \' \s \x50odman is \U0001F51F/\u0031\u0030 \110ello \127orld`,
			expected: []string{"\a", "\b", "\f", "\n", "\r", "\t", "\v", "\\", "\"", "'", " ", "Podman", "is", "🔟/10",
				"Hello", "World"},
		},
		{
			name: "SplitCUnescape don't keep trailing backslash", flags: SplitCUnescape,
			input:         `\a \`,
			expectedError: errUnbalancedEscape,
		},
		{
			name: "SplitCUnescape unsupported escape sequence", flags: SplitCUnescape,
			input:         `\k`,
			expectedError: errUnsupportedEscapeChar,
		},
		{
			name: "SplitUnescapeRelax", flags: SplitUnescapeRelax,
			input:    `\k \z \`,
			expected: []string{"k", "z", "\\"},
		},
		{
			name: "SplitUnescapeSeparators | SplitCUnescape", flags: SplitUnescapeSeparators | SplitCUnescape,
			input:    `hello\ world value other\tvalue`,
			expected: []string{"hello world", "value", "other\tvalue"},
		},
		{
			name:     `No flags "hello world" 'goodbye world'`,
			input:    `"hello world" 'goodbye world'`,
			expected: []string{`"hello`, `world"`, `'goodbye`, `world'`},
		},
		{
			name: `SplitKeepQuote "hello world" "goodbye world"`, flags: SplitKeepQuote,
			input:    `"hello world" 'goodbye world'`,
			expected: []string{`"hello world"`, `'goodbye world'`},
		},
		{
			name: "SplitKeepQuote unbalanced quotes", flags: SplitKeepQuote,
			input:         `"unbalanced quotes`,
			expectedError: errUnbalancedQuotes,
		},
		{
			name: `SplitUnquote "hello world" "goodbye world"`, flags: SplitUnquote,
			input:    `"hello world" 'goodbye world'`,
			expected: []string{`hello world`, `goodbye world`},
		},
		{
			name:  `No flags multiple adjacent separators`,
			input: `hello:::world::goodbye::::world`, separators: ":",
			expected: []string{"hello", "world", "goodbye", "world"},
		},
		{
			name: `SplitDontCoalesceSeparators`, flags: SplitDontCoalesceSeparators,
			input: `hello:::world::goodbye::::world`, separators: ":",
			expected: []string{"hello", "", "", "world", "", "goodbye", "", "", "", "world"},
		},
		{
			name: `SplitRetainEscape 'KEY=val "KEY2=val space" "KEY3=val with \"quotation\""'`, flags: SplitRetainEscape,
			input:    `KEY=val "KEY2=val space" "KEY3=val with \"quotation\""`,
			expected: []string{`KEY=val`, `"KEY2=val`, `space"`, `"KEY3=val`, `with`, `\"quotation\""`},
		},
		{
			name: `SplitRetainEscape 'foo\xbar'`, flags: SplitRetainEscape,
			input:    `foo\xbar`,
			expected: []string{`foo\xbar`},
		},
		{
			name: "SplitRetainSeparators", flags: SplitRetainSeparators,
			input: `a:b`, separators: ":",
			expected: []string{`a`, `b`},
		},
		{
			name: "SplitRetainEscape", flags: SplitRetainSeparators | SplitRetainEscape,
			input: `a\:b`, separators: ":",
			expected: []string{`a\`, `b`},
		},
		{
			name: `SplitDontCoalesceSeparators`, flags: SplitDontCoalesceSeparators,
			input: `:foo\:bar:::waldo:`, separators: ":",
			expected: []string{"", "foo:bar", "", "", "waldo"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			separators := WhitespaceSeparators
			if test.separators != "" {
				separators = test.separators
			}
			output, err := runExtractFirstWord(test.input, separators, test.flags)
			if err != nil {
				require.ErrorIs(t, test.expectedError, err)
			}

			require.Lenf(t, output, len(test.expected), "Expected: %v, Got: %v", test.expected, output)
			for i := range test.expected {
				assert.Equal(t, test.expected[i], output[i])
			}
		})
	}
}

func runExtractFirstWord(input, separators string, flags SplitFlags) ([]string, error) {
	output := make([]string, 0)
	next := input
	for {
		word, remaining, moreWords, err := extractFirstWord(next, separators, flags)
		if err != nil {
			return nil, err
		}

		if !moreWords {
			break
		}

		next = remaining
		output = append(output, word)
	}

	return output, nil
}
