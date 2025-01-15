package assertions

import (
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model"
	generated "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
)

const (
	AssertionPrefix                  = "##"
	NbAssertionComponents            = 6
	NbAssertionComponentsWithErrName = 7
)

func ParseAndReadAssertions(filename string) (model.UnitFile, Assertions, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	strData := string(data)
	lines := strings.Split(strData, "\n")
	slices.Reverse(lines)
	unit, errs := parser.ParseUnitFileString(filename, strData)
	if len(errs) > 0 {
		panic(fmt.Sprintf("errors while parsing unit file: %v", errs))
	}

	var assertions []Assertion
	for _, line := range lines {
		if len(line) == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		if strings.HasPrefix(line, AssertionPrefix) {
			assertion, err := ParseAssertionFromLine(line)
			if err != nil {
				return nil, nil, err
			}
			assertions = append(assertions, assertion)
		} else {
			break
		}
	}

	return unit, assertions, nil
}

var (
	ErrInvalidNbComponents = errors.New("invalid number of components")
	ErrInvalidAssertion    = errors.New("invalid assertion")
	ErrInvalidGroup        = errors.New("invalid group")
	ErrInvalidKey          = errors.New("invalid key")
	ErrInvalidLineNumber   = errors.New("invalid line number")
	ErrInvalidColumnNumber = errors.New("invalid column number")
)

func ParseAssertionFromLine(line string) (Assertion, error) {
	line = strings.TrimPrefix(line, AssertionPrefix)
	components := strings.Fields(line)

	var assertTypeStr, errCategory, errName, group, key, lineNb, columnNb string
	assertTypeStr = components[0]
	errCategory = components[1]
	switch {
	case len(components) < NbAssertionComponents:
		return Assertion{}, fmt.Errorf("%w: expected at least %d components, got %d instead",
			ErrInvalidNbComponents, NbAssertionComponents, len(components))
	case len(components) == NbAssertionComponentsWithErrName:
		errName = components[2]
		group = components[3]
		key = components[4]
		lineNb = components[5]
		columnNb = components[6]
	case len(components) == NbAssertionComponents:
		group = components[2]
		key = components[3]
		lineNb = components[4]
		columnNb = components[5]
	}

	assertType, err := ParseAssertionType(assertTypeStr)
	if err != nil {
		return Assertion{}, fmt.Errorf("%w: %w", ErrInvalidAssertion, err)
	}

	fields, ok := generated.Fields[group]
	if !ok {
		return Assertion{}, fmt.Errorf("%w: %s", ErrInvalidGroup, group)
	}

	if _, ok := fields[key]; !ok {
		return Assertion{}, fmt.Errorf("%w: %s.%s", ErrInvalidKey, group, key)
	}

	errLine, err := strconv.ParseInt(lineNb, 10, strconv.IntSize)
	if err != nil {
		return Assertion{}, fmt.Errorf("%w: %s is not a valid number", ErrInvalidLineNumber, lineNb)
	}

	errCol, err := strconv.ParseInt(columnNb, 10, strconv.IntSize)
	if err != nil {
		return Assertion{}, fmt.Errorf("%w: %s is not a valid number", ErrInvalidColumnNumber, columnNb)
	}

	return Assertion{
		Type:        assertType,
		ErrCategory: errCategory,
		ErrName:     errName,
		Group:       group,
		Key:         key,
		Line:        int(errLine),
		Column:      int(errCol),
	}, nil
}
