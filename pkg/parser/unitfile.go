package parser

import (
	"fmt"
	"math"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"unicode"
)

type unitLine struct {
	key   string
	value string
}

type unitGroup struct {
	name  string
	lines []*unitLine
}

type UnitFile struct {
	groups      []*unitGroup
	groupByName map[string]*unitGroup

	Filename string
	Path     string
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
}

func newParsingError(line int, column int, message string) *ParsingError {
	return &ParsingError{
		message: message,
		Line:    line,
		Column:  column,
	}
}

func newParsingErrorAtLine(line int, message string) *ParsingError {
	return newParsingError(line, 0, message)
}

func (e *ParsingError) Error() string {
	if e.inner != nil {
		return fmt.Sprintf("%s", e.inner.Error())
	}

	return fmt.Sprintf("%s", e.message)
}

func newUnitLine(key string, value string) *unitLine {
	l := &unitLine{
		key:   key,
		value: value,
	}
	return l
}

func (l *unitLine) isKey(key string) bool {
	return l.key == key
}

func (l *unitLine) isEmpty() bool {
	return len(l.value) == 0
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

func (g *unitGroup) prependLine(line *unitLine) {
	n := []*unitLine{line}
	g.lines = append(n, g.lines...)
}

func (g *unitGroup) add(key string, value string) {
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

	f := NewUnitFile()
	f.Path = pathName
	f.Filename = path.Base(pathName)

	parsingErrors := f.Parse(string(data))
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
	return len(line) == 0 || line[0] == '#' || line[0] == ':'
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

func groupNameIsValid(name string) (valid bool, badCharacterIndex int) {
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

func keyNameIsValid(name string) (valid bool, badCharacterIndex int) {
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
		return newParsingError(p.lineNr, badIndex+1, fmt.Sprintf("invalid group name: %s", groupName))
	}

	p.currentGroup = p.file.ensureGroup(groupName)

	return nil
}

func (p *UnitFileParser) parseKeyValuePair(line string) *ParsingError {
	if p.currentGroup == nil {
		return newParsingErrorAtLine(p.lineNr, "key file does not start with a group")
	}

	keyEnd := strings.Index(line, "=")
	valueStart := keyEnd + 1

	// Pull the key name from the line (chomping trailing whitespace)
	for keyEnd > 0 && unicode.IsSpace(rune(line[keyEnd-1])) {
		keyEnd--
	}
	key := line[:keyEnd]
	if valid, badIndex := keyNameIsValid(key); !valid {
		return newParsingError(p.lineNr, badIndex, fmt.Sprintf("invalid key name: %s", key))
	}

	// Pull the value from the line (chugging leading whitespace)

	for valueStart < len(line) && unicode.IsSpace(rune(line[valueStart])) {
		valueStart++
	}

	value := line[valueStart:]

	p.currentGroup.add(key, value)

	return nil
}

func (p *UnitFileParser) parseLine(line string) *ParsingError {
	switch {
	case lineIsGroup(line):
		return p.parseGroup(line)
	case lineIsKeyValuePair(line):
		return p.parseKeyValuePair(line)
	default:
		return newParsingErrorAtLine(p.lineNr, fmt.Sprintf("“%s” is not a key-value pair or group", line))
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
		lineNr: 1,
	}

	data = trimSpacesFromLines(data)

	parsingErrors := make([]ParsingError, 0)
	for len(data) > 0 {
		origdata := data
		nLines := 1
		var line string
		line, data = nextLine(data, 0)

		if !lineIsComment(line) {
			// Handle multi-line continuations
			// Note: This doesn't support comments in the middle of the continuation, which systemd does
			if lineIsKeyValuePair(line) {
				for len(data) > 0 && line[len(line)-1] == '\\' {
					line, data = nextLine(origdata, len(line)+1)
					nLines++
				}
			}
		}

		if lineIsComment(line) {
			continue
		}

		if err := p.parseLine(line); err != nil {
			parsingErrors = append(parsingErrors, *err)
		}

		p.lineNr += nLines
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

func (f *UnitFile) ListKeys(groupName string) []string {
	g, ok := f.groupByName[groupName]
	if !ok {
		return make([]string, 0)
	}

	hash := make(map[string]struct{})
	keys := make([]string, 0, len(g.lines))
	for _, line := range g.lines {
		if _, ok := hash[line.key]; !ok {
			keys = append(keys, line.key)
			hash[line.key] = struct{}{}
		}
	}

	return keys
}

// Look up the last instance of the named key in the group (if any)
// The result can have trailing whitespace, and Raw means it can
// contain line continuations (\ at end of line)
func (f *UnitFile) LookupLastRaw(groupName string, key string) (string, bool) {
	g, ok := f.groupByName[groupName]
	if !ok {
		return "", false
	}

	line := g.findLast(key)
	if line == nil {
		return "", false
	}

	return line.value, true
}

func (f *UnitFile) HasKey(groupName string, key string) bool {
	_, ok := f.LookupLastRaw(groupName, key)
	return ok
}

// Look up the last instance of the named key in the group (if any)
// The result can have trailing whitespace, but line continuations are applied
func (f *UnitFile) LookupLast(groupName string, key string) (string, bool) {
	raw, ok := f.LookupLastRaw(groupName, key)
	if !ok {
		return "", false
	}

	return applyLineContinuation(raw), true
}

// Look up the last instance of the named key in the group (if any)
// The result have no trailing whitespace and line continuations are applied
func (f *UnitFile) Lookup(groupName string, key string) (string, bool) {
	v, ok := f.LookupLast(groupName, key)
	if !ok {
		return "", false
	}

	return strings.Trim(strings.TrimRightFunc(v, unicode.IsSpace), "\""), true
}

// Lookup the last instance of a key and convert the value to a bool
func (f *UnitFile) LookupBoolean(groupName string, key string) (bool, bool) {
	v, ok := f.Lookup(groupName, key)
	if !ok {
		return false, false
	}

	return strings.EqualFold(v, "1") ||
		strings.EqualFold(v, "yes") ||
		strings.EqualFold(v, "true") ||
		strings.EqualFold(v, "on"), true
}

// Lookup the last instance of a key and convert the value to a bool
func (f *UnitFile) LookupBooleanWithDefault(groupName string, key string, defaultValue bool) bool {
	v, ok := f.LookupBoolean(groupName, key)
	if !ok {
		return defaultValue
	}

	return v
}

/* Mimics strol, which is what systemd uses */
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

// Lookup the last instance of a key and convert the value to an int64
func (f *UnitFile) LookupInt(groupName string, key string, defaultValue int64) int64 {
	v, ok := f.Lookup(groupName, key)
	if !ok {
		return defaultValue
	}

	intVal, err := convertNumber(v)

	if err != nil {
		return defaultValue
	}

	return intVal
}

// Lookup the last instance of a key and convert the value to an uint32
func (f *UnitFile) LookupUint32(groupName string, key string, defaultValue uint32) uint32 {
	v := f.LookupInt(groupName, key, int64(defaultValue))
	if v < 0 || v > math.MaxUint32 {
		return defaultValue
	}
	return uint32(v)
}

// Lookup the last instance of a key and convert a uid or a user name to an uint32 uid
func (f *UnitFile) LookupUID(groupName string, key string, defaultValue uint32) (uint32, error) {
	v, ok := f.Lookup(groupName, key)
	if !ok {
		if defaultValue == math.MaxUint32 {
			return 0, fmt.Errorf("no key %s", key)
		}
		return defaultValue, nil
	}

	intVal, err := convertNumber(v)
	if err == nil {
		/* On linux, uids are uint32 values, that can't be (uint32)-1 (== MAXUINT32)*/
		if intVal < 0 || intVal >= math.MaxUint32 {
			return 0, fmt.Errorf("invalid numerical uid '%s'", v)
		}

		return uint32(intVal), nil
	}

	user, err := user.Lookup(v)
	if err != nil {
		return 0, err
	}

	intVal, err = strconv.ParseInt(user.Uid, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint32(intVal), nil
}

// Lookup the last instance of a key and convert a uid or a group name to an uint32 gid
func (f *UnitFile) LookupGID(groupName string, key string, defaultValue uint32) (uint32, error) {
	v, ok := f.Lookup(groupName, key)
	if !ok {
		if defaultValue == math.MaxUint32 {
			return 0, fmt.Errorf("no key %s", key)
		}
		return defaultValue, nil
	}

	intVal, err := convertNumber(v)
	if err == nil {
		/* On linux, uids are uint32 values, that can't be (uint32)-1 (== MAXUINT32)*/
		if intVal < 0 || intVal >= math.MaxUint32 {
			return 0, fmt.Errorf("invalid numerical uid '%s'", v)
		}

		return uint32(intVal), nil
	}

	group, err := user.LookupGroup(v)
	if err != nil {
		return 0, err
	}

	intVal, err = strconv.ParseInt(group.Gid, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint32(intVal), nil
}

// Look up every instance of the named key in the group
// The result can have trailing whitespace, and Raw means it can
// contain line continuations (\ at end of line)
func (f *UnitFile) LookupAllRaw(groupName string, key string) []string {
	g, ok := f.groupByName[groupName]
	if !ok {
		return make([]string, 0)
	}

	values := make([]string, 0)

	for _, line := range g.lines {
		if line.isKey(key) {
			if len(line.value) == 0 {
				// Empty value clears all before
				values = make([]string, 0)
			} else {
				values = append(values, line.value)
			}
		}
	}

	return values
}

// Look up every instance of the named key in the group
// The result can have trailing whitespace, but line continuations are applied
func (f *UnitFile) LookupAll(groupName string, key string) []string {
	values := f.LookupAllRaw(groupName, key)
	for i, raw := range values {
		values[i] = applyLineContinuation(raw)
	}
	return values
}

// Look up every instance of the named key in the group, and for each, split space
// separated words (including handling quoted words) and combine them all into
// one array of words. The split code is compatible with the systemd config_parse_strv().
// This is typically used by systemd keys like "RequiredBy" and "Aliases".
func (f *UnitFile) LookupAllStrv(groupName string, key string) []string {
	res := make([]string, 0)
	values := f.LookupAll(groupName, key)
	for _, value := range values {
		res, _ = splitStringAppend(res, value, WhitespaceSeparators, SplitRetainEscape|SplitUnquote)
	}
	return res
}

// Look up every instance of the named key in the group, and for each, split space
// separated words (including handling quoted words) and combine them all into
// one array of words. The split code is exec-like, and both unquotes and applied
// c-style c escapes.
func (f *UnitFile) LookupAllArgs(groupName string, key string) []string {
	res := make([]string, 0)
	argsv := f.LookupAll(groupName, key)
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
func (f *UnitFile) LookupLastArgs(groupName string, key string) ([]string, bool) {
	execKey, ok := f.LookupLast(groupName, key)
	if ok {
		execArgs, err := splitString(execKey, WhitespaceSeparators, SplitRelax|SplitUnquote|SplitCUnescape)
		if err == nil {
			return execArgs, true
		}
	}
	return nil, false
}

// Look up 'Environment' style key-value keys
func (f *UnitFile) LookupAllKeyVal(groupName string, key string) map[string]string {
	res := make(map[string]string)
	allKeyvals := f.LookupAll(groupName, key)
	for _, keyvals := range allKeyvals {
		assigns, err := splitString(keyvals, WhitespaceSeparators, SplitRelax|SplitUnquote|SplitCUnescape)
		if err == nil {
			for _, assign := range assigns {
				key, value, found := strings.Cut(assign, "=")
				if found {
					res[key] = value
				}
			}
		}
	}
	return res
}
