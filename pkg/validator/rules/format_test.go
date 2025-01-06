package rules

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormat_ParseAndValidate(t *testing.T) {
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
	assert.Nil(t, err)
	assert.Equal(t, "test", format.Value)
	assert.Len(t, format.Options, 0)

	err = format.ParseAndValidate("test@opt1=val1*opt2")
	assert.Nil(t, err)
	assert.Equal(t, "test", format.Value)
	assert.Equal(t, map[string]string{"opt1": "val1", "opt2": "opt2"}, format.Options)

	err = format.ParseAndValidate("bad@opt1*opt2")
	assert.ErrorIs(t, err, ErrInvalidOptions)

	err = format.ParseAndValidate("test@test@test")
	assert.ErrorIs(t, err, ErrInvalidPartLen)

	err = format.ParseAndValidate("test@")
	assert.ErrorIs(t, err, ErrEmptyOpts)

	err = format.ParseAndValidate("test@opt1=val1*")
	assert.ErrorIs(t, err, ErrNoRemainingOpts)
}
