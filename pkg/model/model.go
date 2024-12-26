package model

import (
	"fmt"
)

type LookupFunc struct {
	Name     string
	Multiple bool
}

// TODO: Generate these by downloading the unitfile.go parser file and extracting all
// function declarations starting with 'Lookup'
var (
	UnsupportedLookup        = LookupFunc{Name: "UnsupportedLookup"}
	Lookup                   = LookupFunc{Name: "Lookup"}
	LookupLast               = LookupFunc{Name: "LookupLast"}
	LookupLastRaw            = LookupFunc{Name: "LookupLastRaw"}
	LookupAll                = LookupFunc{Name: "LookupAll", Multiple: true}
	LookupAllRaw             = LookupFunc{Name: "LookupAllRaw", Multiple: true}
	LookupAllStrv            = LookupFunc{Name: "LookupAllStrv", Multiple: true}
	LookupAllArgs            = LookupFunc{Name: "LookupAllArgs", Multiple: true}
	LookupBoolean            = LookupFunc{Name: "LookupBoolean"}
	LookupBooleanWithDefault = LookupFunc{Name: "LookupBooleanWithDefault"}
	LookupInt                = LookupFunc{Name: "LookupInt"}
	LookupUint32             = LookupFunc{Name: "LookupUint32"}
	LookupUID                = LookupFunc{Name: "LookupUID"}
	LookupGID                = LookupFunc{Name: "LookupGID"}
	LookupLastArgs           = LookupFunc{Name: "LookupLastArgs", Multiple: true}
	LookupAllKeyVal          = LookupFunc{Name: "LookupAllKeyVal", Multiple: true}
)

var AllLookupFuncs = map[string]LookupFunc{
	"Lookup":                   Lookup,
	"LookupLast":               LookupLast,
	"LookupLastRaw":            LookupLastRaw,
	"LookupAll":                LookupAll,
	"LookupAllRaw":             LookupAllRaw,
	"LookupAllStrv":            LookupAllStrv,
	"LookupAllArgs":            LookupAllArgs,
	"LookupBoolean":            LookupBoolean,
	"LookupBooleanWithDefault": LookupBooleanWithDefault,
	"LookupInt":                LookupInt,
	"LookupUint32":             LookupUint32,
	"LookupUID":                LookupUID,
	"LookupGID":                LookupGID,
	"LookupLastArgs":           LookupLastArgs,
	"LookupAllKeyVal":          LookupAllKeyVal,
}

type Field struct {
	Group      string
	Key        string
	LookupFunc LookupFunc
}

func (f Field) Multiple() bool {
	return f.LookupFunc.Multiple
}

func (f Field) String() string {
	return fmt.Sprintf("%s.%s", f.Group, f.Key)
}
