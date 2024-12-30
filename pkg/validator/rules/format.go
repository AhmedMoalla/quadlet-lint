package rules

import (
	"fmt"
	"strings"
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
		return fmt.Errorf("'%s' does not match the '%s' format because it is expected to have 2 parts after "+
			"splitting the value with '%s' but got instead %d parts", value, f.Name, f.ValueSeparator, len(split))
	}

	f.Value = split[0]

	if len(split) == 1 { // no options
		return nil
	}

	split = strings.Split(split[1], f.OptionsSeparator)
	options := make(map[string]string, len(split))
	for _, pair := range split {
		kv := strings.Split(pair, "=")
		options[kv[0]] = kv[1]
	}
	f.Options = options

	if f.ValidateOptions != nil {
		if err := f.ValidateOptions(f.Value, f.Options); err != nil {
			return err
		}
	}

	return nil
}
