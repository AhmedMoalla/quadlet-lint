package rules

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormat_ParseAndValidate(t *testing.T) {
	t.Parallel()

	format := Format{
		Name:             "TestFormat",
		ValueSeparator:   "@",
		OptionsSeparator: "*",
		ValidateOptions: func(value string, options map[string]string) error {
			if strings.Contains(value, "bad") {
				return errors.New("bad")
			}
			return nil
		},
	}

	err := format.ParseAndValidate("test")
	require.NoError(t, err)
	assert.Equal(t, "test", format.Value)
	assert.Empty(t, format.Options)

	err = format.ParseAndValidate("test@opt1=val1*opt2")
	require.NoError(t, err)
	assert.Equal(t, "test", format.Value)
	assert.Equal(t, map[string]string{"opt1": "val1", "opt2": "opt2"}, format.Options)

	err = format.ParseAndValidate("bad@opt1*opt2")
	require.ErrorIs(t, err, ErrInvalidOptions)

	err = format.ParseAndValidate("test@test@test")
	require.ErrorIs(t, err, ErrInvalidPartLen)

	err = format.ParseAndValidate("test@")
	require.ErrorIs(t, err, ErrEmptyOpts)

	err = format.ParseAndValidate("test@opt1=val1*")
	require.ErrorIs(t, err, ErrNoRemainingOpts)
}
