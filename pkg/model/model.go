package model

import (
	"fmt"
)

type Field struct {
	Group    string
	Key      string
	Multiple bool
}

func (f Field) String() string {
	return fmt.Sprintf("%s.%s", f.Group, f.Key)
}
