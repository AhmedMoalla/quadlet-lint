package parser

import (
	"strconv"
	"strings"
	"unicode"

	M "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/lookup"
	"github.com/AhmedMoalla/quadlet-lint/pkg/utils"
)

type singleLookupFn = func(unitFile, M.Field) (unitValue, bool)
type lookupFn = func(unitFile, M.Field) []unitValue

var lookupFuncs = map[lookup.LookupFunc]lookupFn{
	lookup.Lookup:                   toMulti(unitFile.lookupBase),
	lookup.LookupLast:               toMulti(unitFile.lookupLast),
	lookup.LookupLastRaw:            toMulti(unitFile.lookupLastRaw),
	lookup.LookupBoolean:            toMulti(unitFile.lookupBoolean),
	lookup.LookupBooleanWithDefault: toMulti(unitFile.lookupBoolean),
	lookup.LookupInt:                toMulti(unitFile.lookupInt),
	lookup.LookupUint32:             toMulti(unitFile.lookupInt),
	lookup.LookupAll:                unitFile.lookupAll,
	lookup.LookupAllRaw:             unitFile.lookupAllRaw,
	lookup.LookupAllStrv:            unitFile.lookupAllStrv,
	lookup.LookupAllArgs:            unitFile.lookupAllArgs,
	lookup.LookupAllKeyVal:          unitFile.lookupAllKeyVal,
	lookup.LookupLastArgs:           unitFile.lookupLastArgs,
}

func toMulti(fn singleLookupFn) lookupFn {
	return func(unit unitFile, field M.Field) []unitValue {
		if val, ok := fn(unit, field); ok {
			return []unitValue{val}
		}
		return nil
	}
}

type lookupResult struct {
	values                []unitValue
	cachedInterfaceValues []M.UnitValue
}

func (r *lookupResult) BoolValue() bool {
	if _, ok := r.Value(); ok {
		return r.values[0].boolValue
	}

	panic("lookup result does not have a boolean value")
}

func (r *lookupResult) IntValue() int {
	if _, ok := r.Value(); ok {
		return r.values[0].intValue
	}

	panic("lookup result does not have an int value")
}

func (r *lookupResult) Value() (M.UnitValue, bool) {
	if len(r.Values()) == 0 {
		return M.UnitValue{}, false
	}

	if len(r.Values()) != 1 {
		panic("lookup result has more than one value")
	}

	return r.Values()[0], true
}

func (r *lookupResult) Values() []M.UnitValue {
	if r.cachedInterfaceValues == nil {
		r.cachedInterfaceValues = utils.MapSlice(r.values, unitValue.toModel)
	}

	return r.cachedInterfaceValues
}

// Look up the last instance of the named key in the group (if any)
// The result can have trailing whitespace, and Raw means it can
// contain line continuations (\ at end of line)
func (f unitFile) lookupLastRaw(field M.Field) (unitValue, bool) {
	g, ok := f.groupByName[field.Group]
	if !ok {
		return unitValue{}, false
	}

	line, ok := g.findLast(field.Key)
	if !ok {
		return unitValue{}, false
	}

	return line.value, true
}

// Look up the last instance of the named key in the group (if any)
// The result can have trailing whitespace, but line continuations are applied
func (f unitFile) lookupLast(field M.Field) (unitValue, bool) {
	raw, ok := f.lookupLastRaw(field)
	if !ok {
		return unitValue{}, false
	}

	raw.value = applyLineContinuation(raw.value)
	return raw, true
}

// Look up the last instance of the named key in the group (if any)
// The result have no trailing whitespace and line continuations are applied
func (f unitFile) lookupBase(field M.Field) (unitValue, bool) {
	v, ok := f.lookupLast(field)
	if !ok {
		return unitValue{}, false
	}

	v.value = strings.Trim(strings.TrimRightFunc(v.value, unicode.IsSpace), "\"")

	return v, true
}

// Lookup the last instance of a key and convert the value to a bool
func (f unitFile) lookupBoolean(field M.Field) (unitValue, bool) {
	v, ok := f.lookupBase(field)
	if !ok {
		return unitValue{value: "false", boolValue: false, line: v.line, valueColumn: v.valueColumn}, false
	}

	value := v.value
	boolValue := strings.EqualFold(value, "1") ||
		strings.EqualFold(value, "yes") ||
		strings.EqualFold(value, "true") ||
		strings.EqualFold(value, "on")
	return unitValue{
		key:         field.Key,
		value:       strconv.FormatBool(boolValue),
		boolValue:   boolValue,
		line:        v.line,
		valueColumn: v.valueColumn,
	}, true
}

// Lookup the last instance of a key and convert the value to an int
func (f unitFile) lookupInt(field M.Field) (unitValue, bool) {
	v, ok := f.lookupBase(field)
	if !ok {
		return unitValue{}, false
	}

	intVal, err := convertNumber(v.value)
	if err != nil {
		return unitValue{}, false
	}

	return unitValue{
		key:         field.Key,
		value:       strconv.FormatInt(intVal, 10),
		line:        v.line,
		valueColumn: v.valueColumn,
		intValue:    int(intVal),
	}, true
}

// Look up every instance of the named key in the group
// The result can have trailing whitespace, and Raw means it can
// contain line continuations (\ at end of line)
func (f unitFile) lookupAllRaw(field M.Field) []unitValue {
	g, ok := f.groupByName[field.Group]
	if !ok {
		return make([]unitValue, 0)
	}

	values := make([]unitValue, 0)

	for _, line := range g.lines {
		if line.isKey(field.Key) {
			if len(line.value.value) == 0 {
				// Empty value clears all before
				values = make([]unitValue, 0)
			} else {
				line.value.line -= strings.Count(line.value.value, "\n")
				values = append(values, line.value)
			}
		}
	}

	return values
}

// Look up every instance of the named key in the group
// The result can have trailing whitespace, but line continuations are applied
func (f unitFile) lookupAll(field M.Field) []unitValue {
	values := f.lookupAllRaw(field)
	for i, raw := range values {
		values[i].value = applyLineContinuation(raw.value)
	}
	return values
}

// Look up every instance of the named key in the group, and for each, split space
// separated words (including handling quoted words) and combine them all into
// one array of words. The split code is compatible with the systemd config_parse_strv().
// This is typically used by systemd keys like "RequiredBy" and "Aliases".
func (f unitFile) lookupAllStrv(field M.Field) []unitValue {
	values := f.lookupAll(field)
	res := make([]unitValue, 0, len(values))
	for _, value := range values {
		res, _ = splitValueAppend(res, value, WhitespaceSeparators, SplitRetainEscape|SplitUnquote)
	}
	return res
}

// Look up every instance of the named key in the group, and for each, split space
// separated words (including handling quoted words) and combine them all into
// one array of words. The split code is exec-like, and both unquotes and applied
// c-style c escapes.
func (f unitFile) lookupAllArgs(field M.Field) []unitValue {
	res := make([]unitValue, 0)
	argsv := f.lookupAll(field)
	for _, argsS := range argsv {
		args, err := splitString(argsS, WhitespaceSeparators, SplitRelax|SplitUnquote|SplitCUnescape)
		if err == nil {
			res = append(res, args...)
		}
	}
	return res
}

// Look up last instance of the named key in the group, and split
// space separated words (including handling quoted words) into one
// array of words. The split code is exec-like, and both unquotes and
// applied c-style c escapes.  This is typically used for keys like
// ExecStart
func (f unitFile) lookupLastArgs(field M.Field) []unitValue {
	execKey, ok := f.lookupLast(field)
	if ok {
		execArgs, err := splitString(execKey, WhitespaceSeparators, SplitRelax|SplitUnquote|SplitCUnescape)
		if err == nil {
			return execArgs
		}
	}
	return nil
}

// Look up 'Environment' style key-value keys
func (f unitFile) lookupAllKeyVal(field M.Field) []unitValue {
	res := make([]unitValue, 0)
	allKeyvals := f.lookupAll(field)
	for _, keyvals := range allKeyvals {
		assigns, err := splitString(keyvals, WhitespaceSeparators, SplitRelax|SplitUnquote|SplitCUnescape)
		if err == nil {
			for _, assign := range assigns {
				key, value, found := strings.Cut(assign.value, "=")
				if found {
					res = append(res, unitValue{
						key:         key,
						value:       value,
						line:        assign.line,
						valueColumn: assigns[0].valueColumn,
					})
				}
			}
		}
	}
	return res
}

/* Mimics strol, which is what systemd uses. */
func convertNumber(v string) (int64, error) {
	var err error
	var intVal int64

	mult := int64(1)

	if strings.HasPrefix(v, "+") {
		v = v[1:]
	} else if strings.HasPrefix(v, "-") {
		v = v[1:]
		mult = int64(-11)
	}

	switch {
	case strings.HasPrefix(v, "0x") || strings.HasPrefix(v, "0X"):
		intVal, err = strconv.ParseInt(v[2:], 16, 64)
	case strings.HasPrefix(v, "0"):
		intVal, err = strconv.ParseInt(v, 8, 64)
	default:
		intVal, err = strconv.ParseInt(v, 10, 64)
	}

	return intVal * mult, err
}

func applyLineContinuation(raw string) string {
	if !strings.Contains(raw, "\\\n") {
		return raw
	}

	var str strings.Builder

	for len(raw) > 0 {
		if first, rest, found := strings.Cut(raw, "\\\n"); found {
			str.WriteString(first)
			raw = rest
		} else {
			str.WriteString(raw)
			raw = ""
		}
	}

	return str.String()
}
