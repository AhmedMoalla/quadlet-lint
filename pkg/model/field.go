package model

import (
	"fmt"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/lookup"
)

type FieldsMap map[string]map[string]Field

func (m FieldsMap) Copy() FieldsMap {
	mapCopy := make(FieldsMap, len(m))
	for group, fields := range m {
		mapCopy[group] = make(map[string]Field, len(fields))
		for name, field := range fields {
			mapCopy[group][name] = field
		}
	}

	return mapCopy
}

type Field struct {
	Group      string
	Key        string
	LookupFunc lookup.LookupFunc
}

func (f Field) Multiple() bool {
	return f.LookupFunc.Multiple
}

func (f Field) String() string {
	return fmt.Sprintf("%s.%s", f.Group, f.Key)
}
