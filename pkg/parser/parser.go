package parser

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model"
	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/lookup"
)

type unitLine struct {
	key   string
	value UnitValue
}

type UnitKey struct {
	Key  string
	Line int
}

type UnitValue struct {
	Key    string
	Value  string
	Line   int
	Column int

	intValue     int
	booleanValue bool
}

func (u *UnitValue) String() string {
	return u.Value
}

type unitGroup struct {
	name  string
	lines []*unitLine
}

type UnitType struct {
	Name string
	Ext  string
}

var (
	UnitTypeContainer = UnitType{Name: "container", Ext: ".container"}
	UnitTypeVolume    = UnitType{Name: "volume", Ext: ".volume"}
	UnitTypeKube      = UnitType{Name: "kube", Ext: ".kube"}
	UnitTypeNetwork   = UnitType{Name: "network", Ext: ".network"}
	UnitTypeImage     = UnitType{Name: "image", Ext: ".image"}
	UnitTypeBuild     = UnitType{Name: "build", Ext: ".build"}
	UnitTypePod       = UnitType{Name: "pod", Ext: ".pod"}
)

type UnitFile struct {
	groups      []*unitGroup
	groupByName map[string]*unitGroup

	Filename string
	UnitType UnitType
}

type LookupResult struct {
	Values []UnitValue
}

func (r *LookupResult) BooleanValue() bool {
	if val := r.Value(); val != nil {
		return val.booleanValue
	}

	panic("lookup result does not have a boolean value")
}

func (r *LookupResult) IntValue() int {
	if val := r.Value(); val != nil {
		return val.intValue
	}

	panic("lookup result does not have an int value")
}

func (r *LookupResult) Value() *UnitValue {
	if len(r.Values) == 0 {
		return nil
	}

	if len(r.Values) != 1 {
		panic("lookup result has more than one value")
	}

	return &r.Values[0]
}

func toMulti(fn SingleLookupFn) LookupFn {
	return func(unit *UnitFile, field model.Field) []UnitValue {
		if val, ok := fn(unit, field); ok {
			return []UnitValue{val}
		}
		return nil
	}
}

type SingleLookupFn = func(*UnitFile, model.Field) (UnitValue, bool)
type LookupFn = func(*UnitFile, model.Field) []UnitValue

var lookupFuncs = map[lookup.LookupFunc]LookupFn{
	lookup.Lookup:                   toMulti((*UnitFile).lookupBase),
	lookup.LookupLast:               toMulti((*UnitFile).lookupLast),
	lookup.LookupLastRaw:            toMulti((*UnitFile).lookupLastRaw),
	lookup.LookupBoolean:            toMulti((*UnitFile).lookupBoolean),
	lookup.LookupBooleanWithDefault: toMulti((*UnitFile).lookupBoolean),
	lookup.LookupInt:                toMulti((*UnitFile).lookupInt),
	lookup.LookupUint32:             toMulti((*UnitFile).lookupInt),
	lookup.LookupAll:                (*UnitFile).lookupAll,
	lookup.LookupAllRaw:             (*UnitFile).lookupAllRaw,
	lookup.LookupAllStrv:            (*UnitFile).lookupAllStrv,
	lookup.LookupAllArgs:            (*UnitFile).lookupAllArgs,
	lookup.LookupAllKeyVal:          (*UnitFile).lookupAllKeyVal,
	lookup.LookupLastArgs:           (*UnitFile).lookupLastArgs,
}

func (f *UnitFile) Lookup(field model.Field) (LookupResult, bool) {
	if fn, ok := lookupFuncs[field.LookupFunc]; ok {
		values := fn(f, field)
		return LookupResult{Values: values}, len(values) > 0
	}

	panic(fmt.Sprintf("lookup mode %s is not supported for field %s", field.LookupFunc.Name, field.Key))
}

type UnitFileParser struct {
	file *UnitFile

	currentGroup *unitGroup
	lineNr       int
}

type ParsingError struct {
	inner   error
	message string
	Line    int
	Column  int
	Group   string
	Key     string
}

func newParsingError(line int, column int, group, key, message string) *ParsingError {
	return &ParsingError{
		message: message,
		Line:    line,
		Column:  column,
		Group:   group,
		Key:     key,
	}
}

func newParsingErrorAtLine(line int, group, key, message string) *ParsingError {
	return newParsingError(line, 0, group, key, message)
}

func (e *ParsingError) Error() string {
	if e.inner != nil {
		return e.inner.Error()
	}

	return e.message
}

func newUnitLine(key string, value UnitValue) *unitLine {
	l := &unitLine{
		key:   key,
		value: value,
	}
	return l
}

func (l *unitLine) isKey(key string) bool {
	return l.key == key
}

func newUnitGroup(name string) *unitGroup {
	g := &unitGroup{
		name:  name,
		lines: make([]*unitLine, 0),
	}
	return g
}

func (g *unitGroup) addLine(line *unitLine) {
	g.lines = append(g.lines, line)
}

func (g *unitGroup) add(key string, value UnitValue) {
	g.addLine(newUnitLine(key, value))
}

func (g *unitGroup) findLast(key string) *unitLine {
	for i := len(g.lines) - 1; i >= 0; i-- {
		l := g.lines[i]
		if l.isKey(key) {
			return l
		}
	}

	return nil
}

// Create an empty unit file, with no filename or path
func NewUnitFile() *UnitFile {
	f := &UnitFile{
		groups:      make([]*unitGroup, 0),
		groupByName: make(map[string]*unitGroup),
	}

	return f
}

// Load a unit file from disk, remembering the path and filename
func ParseUnitFile(pathName string) (*UnitFile, []ParsingError) {
	data, e := os.ReadFile(pathName)
	if e != nil {
		return nil, []ParsingError{{inner: e}}
	}

	return ParseUnitFileString(pathName, string(data))
}

func ParseUnitFileString(pathName, content string) (*UnitFile, []ParsingError) {
	f := NewUnitFile()
	f.Filename = path.Base(pathName)
	ext := path.Ext(pathName)
	f.UnitType = UnitType{Name: ext[1:], Ext: ext}

	parsingErrors := f.Parse(content)
	if len(parsingErrors) > 0 {
		return nil, parsingErrors
	}

	return f, []ParsingError{}
}

func (f *UnitFile) ensureGroup(groupName string) *unitGroup {
	if g, ok := f.groupByName[groupName]; ok {
		return g
	}

	g := newUnitGroup(groupName)
	f.groups = append(f.groups, g)
	f.groupByName[groupName] = g

	return g
}

func lineIsComment(line string) bool {
	return len(line) == 0 || line[0] == '#' || line[0] == ';'
}

func lineIsGroup(line string) bool {
	if len(line) == 0 {
		return false
	}

	if line[0] != '[' {
		return false
	}

	end := strings.Index(line, "]")
	if end == -1 {
		return false
	}

	// silently accept whitespace after the ]
	for i := end + 1; i < len(line); i++ {
		if line[i] != ' ' && line[i] != '\t' {
			return false
		}
	}

	return true
}

func lineIsKeyValuePair(line string) bool {
	if len(line) == 0 {
		return false
	}

	p := strings.IndexByte(line, '=')
	if p == -1 {
		return false
	}

	// Key must be non-empty
	if p == 0 {
		return false
	}

	return true
}

// groupNameIsValid checks if a group's name is valid and returns the position where the group name is invalid
func groupNameIsValid(name string) (bool, int) {
	if len(name) == 0 {
		return false, 0
	}

	for index, c := range name {
		if c == ']' || c == '[' || unicode.IsControl(c) {
			return false, index
		}
	}

	return true, -1
}

// keyNameIsValid checks if a key's name is valid and returns the position where the key name is invalid
func keyNameIsValid(name string) (bool, int) {
	if len(name) == 0 {
		return false, 0
	}

	for index, c := range name {
		if c == '=' {
			return false, index
		}
	}

	// No leading/trailing space
	if name[0] == ' ' {
		return false, 0
	}

	if name[len(name)-1] == ' ' {
		return false, len(name) - 1
	}

	return true, -1
}

func (p *UnitFileParser) parseGroup(line string) *ParsingError {
	end := strings.Index(line, "]")

	groupName := line[1:end]

	if valid, badIndex := groupNameIsValid(groupName); !valid {
		return newParsingError(p.lineNr, badIndex+1, p.currentGroup.name, "", "invalid group name: "+groupName)
	}

	p.currentGroup = p.file.ensureGroup(groupName)

	return nil
}

func (p *UnitFileParser) parseKeyValuePair(line string) *ParsingError {
	if p.currentGroup == nil {
		return newParsingErrorAtLine(p.lineNr, "", "", "key file does not start with a group")
	}

	keyEnd := strings.Index(line, "=")
	valueStart := keyEnd + 1

	// Pull the key name from the line (chomping trailing whitespace)
	for keyEnd > 0 && unicode.IsSpace(rune(line[keyEnd-1])) {
		keyEnd--
	}
	key := line[:keyEnd]
	if valid, badIndex := keyNameIsValid(key); !valid {
		return newParsingError(p.lineNr, badIndex, p.currentGroup.name, key, "invalid key name: "+key)
	}

	// Pull the value from the line (chugging leading whitespace)

	for valueStart < len(line) && unicode.IsSpace(rune(line[valueStart])) {
		valueStart++
	}

	value := line[valueStart:]

	if len(value) == 0 {
		return newParsingError(p.lineNr, valueStart, p.currentGroup.name, key,
			fmt.Sprintf("key '%s' in group '%s' has an empty value", key, p.currentGroup.name))
	}

	p.currentGroup.add(key, UnitValue{
		Key:    key,
		Value:  value,
		Line:   p.lineNr,
		Column: valueStart,
	})

	return nil
}

func (p *UnitFileParser) parseLine(line string) *ParsingError {
	switch {
	case lineIsGroup(line):
		return p.parseGroup(line)
	case lineIsKeyValuePair(line):
		return p.parseKeyValuePair(line)
	default:
		return newParsingErrorAtLine(p.lineNr, p.currentGroup.name, "", fmt.Sprintf("“%s” is not a key-value pair or group", line))
	}
}

func nextLine(data string, afterPos int) (string, string) {
	rest := data[afterPos:]
	if i := strings.Index(rest, "\n"); i >= 0 {
		return strings.TrimSpace(data[:i+afterPos]), data[i+afterPos+1:]
	}
	return data, ""
}

func trimSpacesFromLines(data string) string {
	lines := strings.Split(data, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

// Parse an already loaded unit file (in the form of a string)
func (f *UnitFile) Parse(data string) []ParsingError {
	p := &UnitFileParser{
		file:   f,
		lineNr: 0,
	}

	data = trimSpacesFromLines(data)

	parsingErrors := make([]ParsingError, 0)
	for len(data) > 0 {
		origdata := data
		var line string
		line, data = nextLine(data, 0)
		p.lineNr++

		if lineIsComment(line) {
			continue
		}

		// Handle multi-line continuations
		// Note: This doesn't support comments in the middle of the continuation, which systemd does
		if lineIsKeyValuePair(line) {
			for len(data) > 0 && line[len(line)-1] == '\\' {
				line, data = nextLine(origdata, len(line)+1)
				p.lineNr++
			}
		}

		if err := p.parseLine(line); err != nil {
			parsingErrors = append(parsingErrors, *err)
		}
	}

	if p.currentGroup == nil {
		// For files without groups, add an empty group name used only for initial comments
		p.currentGroup = p.file.ensureGroup("")
	}

	return parsingErrors
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

func (f *UnitFile) HasGroup(groupName string) bool {
	_, ok := f.groupByName[groupName]
	return ok
}

func (f *UnitFile) ListGroups() []string {
	groups := make([]string, len(f.groups))
	for i, group := range f.groups {
		groups[i] = group.name
	}
	return groups
}

func (f *UnitFile) ListKeys(groupName string) []UnitKey {
	g, ok := f.groupByName[groupName]
	if !ok {
		return make([]UnitKey, 0)
	}

	hash := make(map[string]struct{})
	keys := make([]UnitKey, 0, len(g.lines))
	for _, line := range g.lines {
		if _, ok := hash[line.key]; !ok {
			keys = append(keys, UnitKey{Key: line.key, Line: line.value.Line})
			hash[line.key] = struct{}{}
		}
	}

	return keys
}

// Look up the last instance of the named key in the group (if any)
// The result can have trailing whitespace, and Raw means it can
// contain line continuations (\ at end of line)
func (f *UnitFile) lookupLastRaw(field model.Field) (UnitValue, bool) {
	g, ok := f.groupByName[field.Group]
	if !ok {
		return UnitValue{}, false
	}

	line := g.findLast(field.Key)
	if line == nil {
		return UnitValue{}, false
	}

	return line.value, true
}

func (f *UnitFile) HasKey(field model.Field) bool {
	_, ok := f.lookupLastRaw(field)
	return ok
}

// Look up the last instance of the named key in the group (if any)
// The result can have trailing whitespace, but line continuations are applied
func (f *UnitFile) lookupLast(field model.Field) (UnitValue, bool) {
	raw, ok := f.lookupLastRaw(field)
	if !ok {
		return UnitValue{}, false
	}

	raw.Value = applyLineContinuation(raw.Value)
	return raw, true
}

// Look up the last instance of the named key in the group (if any)
// The result have no trailing whitespace and line continuations are applied
func (f *UnitFile) lookupBase(field model.Field) (UnitValue, bool) {
	v, ok := f.lookupLast(field)
	if !ok {
		return UnitValue{}, false
	}

	v.Value = strings.Trim(strings.TrimRightFunc(v.Value, unicode.IsSpace), "\"")

	return v, true
}

// Lookup the last instance of a key and convert the value to a bool
func (f *UnitFile) lookupBoolean(field model.Field) (UnitValue, bool) {
	v, ok := f.lookupBase(field)
	if !ok {
		return UnitValue{Value: "false", booleanValue: false, Line: v.Line, Column: v.Column}, false
	}

	value := v.Value
	booleanValue := strings.EqualFold(value, "1") ||
		strings.EqualFold(value, "yes") ||
		strings.EqualFold(value, "true") ||
		strings.EqualFold(value, "on")
	return UnitValue{
		Value:        strconv.FormatBool(booleanValue),
		booleanValue: booleanValue,
		Line:         v.Line,
		Column:       v.Column,
	}, true
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

// Lookup the last instance of a key and convert the value to an int
func (f *UnitFile) lookupInt(field model.Field) (UnitValue, bool) {
	v, ok := f.lookupBase(field)
	if !ok {
		return UnitValue{}, false
	}

	intVal, err := convertNumber(v.Value)
	if err != nil {
		return UnitValue{}, false
	}

	return UnitValue{
		Key:      field.Key,
		Value:    strconv.FormatInt(intVal, 10),
		Line:     v.Line,
		Column:   v.Column,
		intValue: int(intVal),
	}, true
}

// Look up every instance of the named key in the group
// The result can have trailing whitespace, and Raw means it can
// contain line continuations (\ at end of line)
func (f *UnitFile) lookupAllRaw(field model.Field) []UnitValue {
	g, ok := f.groupByName[field.Group]
	if !ok {
		return make([]UnitValue, 0)
	}

	values := make([]UnitValue, 0)

	for _, line := range g.lines {
		if line.isKey(field.Key) {
			if len(line.value.Value) == 0 {
				// Empty value clears all before
				values = make([]UnitValue, 0)
			} else {
				values = append(values, line.value)
			}
		}
	}

	return values
}

// Look up every instance of the named key in the group
// The result can have trailing whitespace, but line continuations are applied
func (f *UnitFile) lookupAll(field model.Field) []UnitValue {
	values := f.lookupAllRaw(field)
	for i, raw := range values {
		values[i].Value = applyLineContinuation(raw.Value)
	}
	return values
}

// Look up every instance of the named key in the group, and for each, split space
// separated words (including handling quoted words) and combine them all into
// one array of words. The split code is compatible with the systemd config_parse_strv().
// This is typically used by systemd keys like "RequiredBy" and "Aliases".
func (f *UnitFile) lookupAllStrv(field model.Field) []UnitValue {
	values := f.lookupAll(field)
	res := make([]UnitValue, 0, len(values))
	for _, value := range values {
		res, _ = splitValueAppend(res, value, WhitespaceSeparators, SplitRetainEscape|SplitUnquote)
	}
	return res
}

// Look up every instance of the named key in the group, and for each, split space
// separated words (including handling quoted words) and combine them all into
// one array of words. The split code is exec-like, and both unquotes and applied
// c-style c escapes.
func (f *UnitFile) lookupAllArgs(field model.Field) []UnitValue {
	res := make([]UnitValue, 0)
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
func (f *UnitFile) lookupLastArgs(field model.Field) []UnitValue {
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
func (f *UnitFile) lookupAllKeyVal(field model.Field) []UnitValue {
	res := make([]UnitValue, 0)
	allKeyvals := f.lookupAll(field)
	for _, keyvals := range allKeyvals {
		assigns, err := splitString(keyvals, WhitespaceSeparators, SplitRelax|SplitUnquote|SplitCUnescape)
		if err == nil {
			for _, assign := range assigns {
				key, value, found := strings.Cut(assign.Value, "=")
				if found {
					res = append(res, UnitValue{
						Key:    key,
						Value:  value,
						Line:   assign.Line,
						Column: assign.Column,
					})
				}
			}
		}
	}
	return res
}

func (f *UnitFile) HasValue(field model.Field) bool {
	value, found := f.lookupBase(field)
	return found && len(value.Value) > 0
}
