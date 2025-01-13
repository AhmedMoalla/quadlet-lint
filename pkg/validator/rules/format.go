package rules

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidPartLen  = errors.New("invalid parts length")
	ErrEmptyOpts       = errors.New("no options after separator")
	ErrNoRemainingOpts = errors.New("no remaining options after separator")
	ErrInvalidOptions  = errors.New("invalid options")
)

// Format is the format that a value of a given key has
// For example: the Network key has the format: mode[:options,...] so the Format would be:
// - ValueSeparator: ":"
// - OptionsSeparator: ","
type Format struct {
	Name             string // Name of the Format
	ValueSeparator   string // ValueSeparator is the separator between the value and its options
	OptionsSeparator string // OptionsSeparator is the separator between the options

	Value string // Value is the value before the ValueSeparator. Populated after calling ParseAndValidate.
	// Options are the options after the ValueSeparator split by the OptionsSeparator.
	// Populated after calling ParseAndValidate.
	Options map[string]string

	ValidateOptions func(value string, options map[string]string) error
}

func (f *Format) ParseAndValidate(value string) error {
	split := strings.Split(value, f.ValueSeparator)
	if len(split) == 0 || len(split) > 2 {
		return fmt.Errorf("%w: '%s' does not match the '%s' format because it is expected to have 2 parts after "+
			"splitting the value with '%s' but got instead %d parts", ErrInvalidPartLen, value, f.Name, f.ValueSeparator,
			len(split))
	}

	f.Value = split[0]

	if len(split) == 1 { // no options
		return nil
	}

	if len(split[1]) == 0 { // empty options
		return fmt.Errorf("%w: '%s' does not match the '%s' format because no options were found after "+
			"the value separator '%s'", ErrEmptyOpts, value, f.Name, f.ValueSeparator)
	}

	split = strings.Split(split[1], f.OptionsSeparator)
	options := make(map[string]string, len(split))
	for _, pair := range split {
		kv := strings.Split(pair, "=")
		switch {
		case len(kv) == 1 && len(kv[0]) > 0:
			options[kv[0]] = kv[0]
		case len(kv) == 2: //nolint:mnd
			options[kv[0]] = kv[1]
		default:
			return fmt.Errorf("%w: '%s' does not match the '%s' format because no remaining options were found after "+
				"the options separator '%s'", ErrNoRemainingOpts, value, f.Name, f.OptionsSeparator)
		}
	}
	f.Options = options

	if f.ValidateOptions != nil {
		if err := f.ValidateOptions(f.Value, f.Options); err != nil {
			return errors.Join(err, ErrInvalidOptions)
		}
	}

	return nil
}
