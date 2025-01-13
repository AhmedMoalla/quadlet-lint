package parser

import (
	"fmt"

	M "github.com/AhmedMoalla/quadlet-lint/pkg/model"
)

type unitFile struct {
	groups      []*unitGroup
	groupByName map[string]*unitGroup

	filename string
	unitType M.UnitType
}

func newUnitFile(filename string, unitType M.UnitType) unitFile {
	return unitFile{
		groups:      make([]*unitGroup, 0),
		groupByName: make(map[string]*unitGroup),
		filename:    filename,
		unitType:    unitType,
	}
}

type unitGroup struct {
	name  string
	lines []unitLine
}

func newUnitGroup(name string) *unitGroup {
	return &unitGroup{
		name:  name,
		lines: make([]unitLine, 0),
	}
}

func (g *unitGroup) addLine(line unitLine) {
	g.lines = append(g.lines, line)
}

func (g *unitGroup) add(key string, value unitValue) {
	g.addLine(newUnitLine(key, value))
}

func (g *unitGroup) findLast(key string) (unitLine, bool) {
	for i := len(g.lines) - 1; i >= 0; i-- {
		l := g.lines[i]
		if l.isKey(key) {
			return l, true
		}
	}

	return unitLine{}, false
}

type unitLine struct {
	key   string
	value unitValue
}

func newUnitLine(key string, value unitValue) unitLine {
	return unitLine{
		key:   key,
		value: value,
	}
}

func (l *unitLine) isKey(key string) bool {
	return l.key == key
}

type unitValue struct {
	key         string
	line        int
	valueColumn int

	value     string
	intValue  int
	boolValue bool
}

func (v unitValue) toModel() M.UnitValue {
	return M.UnitValue{
		Key:    v.key,
		Value:  v.value,
		Line:   v.line,
		Column: v.valueColumn,
	}
}

func (f unitFile) FileName() string {
	return f.filename
}

func (f unitFile) UnitType() M.UnitType {
	return f.unitType
}

func (f unitFile) Lookup(field M.Field) (M.LookupResult, bool) {
	if fn, ok := lookupFuncs[field.LookupFunc]; ok {
		values := fn(f, field)
		return lookupResult{values: values}, len(values) > 0
	}

	panic(fmt.Sprintf("lookup mode %s is not supported for field %s", field.LookupFunc.Name, field.Key))
}

func (f unitFile) HasGroup(groupName string) bool {
	_, ok := f.groupByName[groupName]
	return ok
}

func (f unitFile) ListGroups() []string {
	groups := make([]string, len(f.groups))
	for i, group := range f.groups {
		groups[i] = group.name
	}
	return groups
}

func (f unitFile) ListKeys(groupName string) []M.UnitKey {
	g, ok := f.groupByName[groupName]
	if !ok {
		return make([]M.UnitKey, 0)
	}

	hash := make(map[string]struct{})
	keys := make([]M.UnitKey, 0, len(g.lines))
	for _, line := range g.lines {
		if _, ok := hash[line.key]; !ok {
			keys = append(keys, M.UnitKey{Key: line.key, Line: line.value.line})
			hash[line.key] = struct{}{}
		}
	}

	return keys
}

func (f unitFile) HasValue(field M.Field) bool {
	value, found := f.lookupBase(field)
	return found && len(value.value) > 0
}

func (f unitFile) HasKey(field M.Field) bool {
	_, ok := f.lookupLastRaw(field)
	return ok
}
