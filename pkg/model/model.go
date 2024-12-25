package model

import (
	"fmt"
)

type LookupMode string

const (
	LookupModeBase    LookupMode = "Lookup"
	LookupModeLast    LookupMode = "LookupLast"
	LookupModeLastRaw LookupMode = "LookupLastRaw"
	LookupModeAll     LookupMode = "LookupAll"
	LookupModeAllRaw  LookupMode = "LookupAllRaw"
	LookupModeAllStrv LookupMode = "LookupAllStrv"
	LookupModeAllArgs LookupMode = "LookupAllArgs"
)

type Field struct {
	Group      string
	Key        string
	Multiple   bool
	LookupMode LookupMode
}

func (f Field) String() string {
	return fmt.Sprintf("%s.%s", f.Group, f.Key)
}
