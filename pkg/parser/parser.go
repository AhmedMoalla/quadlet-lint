package parser

import (
	"fmt"
	"os"
	"path"
	"strings"
	"unicode"

	M "github.com/AhmedMoalla/quadlet-lint/pkg/model"
)

type unitFileParser struct {
	file *unitFile

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

// Load a unit file from disk, remembering the path and filename
func ParseUnitFile(pathName string) (M.UnitFile, []ParsingError) {
	data, e := os.ReadFile(pathName)
	if e != nil {
		return nil, []ParsingError{{inner: e}}
	}

	return ParseUnitFileString(pathName, string(data))
}

func ParseUnitFileString(pathName, content string) (M.UnitFile, []ParsingError) {
	filename := path.Base(pathName)
	ext := path.Ext(pathName)
	unitType := M.UnitType{Name: ext[1:], Ext: ext}
	f := newUnitFile(filename, unitType)

	parsingErrors := parse(&f, content)
	if len(parsingErrors) > 0 {
		return nil, parsingErrors
	}

	return f, []ParsingError{}
}

// parse an already loaded unit file (in the form of a string)
func parse(f *unitFile, data string) []ParsingError {
	p := &unitFileParser{
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

	return parsingErrors
}

func nextLine(data string, afterPos int) (string, string) {
	rest := data[afterPos:]
	if i := strings.Index(rest, "\n"); i >= 0 {
		return strings.TrimSpace(data[:i+afterPos]), data[i+afterPos+1:]
	}
	return data, ""
}

func (p *unitFileParser) parseLine(line string) *ParsingError {
	switch {
	case lineIsGroup(line):
		return p.parseGroup(line)
	case lineIsKeyValuePair(line):
		return p.parseKeyValuePair(line)
	default:
		return newParsingErrorAtLine(p.lineNr, p.currentGroup.String(), line,
			fmt.Sprintf("“%s” is not a key-value pair or group", line))
	}
}

func (p *unitFileParser) parseGroup(line string) *ParsingError {
	end := strings.Index(line, "]")

	groupName := line[1:end]

	if valid, badIndex := groupNameIsValid(groupName); !valid {
		return newParsingError(p.lineNr, badIndex+1, groupName, "", "invalid group name: "+groupName)
	}

	p.currentGroup = ensureGroup(p.file, groupName)

	return nil
}

func (p *unitFileParser) parseKeyValuePair(line string) *ParsingError {
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

	p.currentGroup.add(key, unitValue{
		key:         key,
		value:       value,
		line:        p.lineNr,
		valueColumn: valueStart,
	})

	return nil
}

func ensureGroup(f *unitFile, groupName string) *unitGroup {
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

func trimSpacesFromLines(data string) string {
	lines := strings.Split(data, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}
