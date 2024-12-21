package service

import (
	P "github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

type Service struct {
	KillMode []V.Rule
	Type     []V.Rule
}

var (
	// Service Group fields
	KillMode = P.Field{Group: "Service", Key: "KillMode"}
	Type     = P.Field{Group: "Service", Key: "Type"}
)
