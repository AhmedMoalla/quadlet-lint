package model

import (
	"fmt"

	"github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/lookup"
)

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
