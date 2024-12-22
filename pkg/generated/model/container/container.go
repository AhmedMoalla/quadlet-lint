package container

import (
	P "github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

type Container struct {
	Rootfs         []V.Rule
	Image          []V.Rule
	Network        []V.Rule
	Volume         []V.Rule
	Mount          []V.Rule
	Pod            []V.Rule
	Group          []V.Rule
	RemapUid       []V.Rule
	RemapGid       []V.Rule
	RemapUsers     []V.Rule
	ExposeHostPort []V.Rule
}

var (
	// Container Group fields
	Image          = P.Field{Group: "Container", Key: "Image"}
	Rootfs         = P.Field{Group: "Container", Key: "Rootfs"}
	Network        = P.Field{Group: "Container", Key: "Network", Multiple: true}
	Volume         = P.Field{Group: "Container", Key: "Volume"}
	Mount          = P.Field{Group: "Container", Key: "Mount"}
	Pod            = P.Field{Group: "Container", Key: "Pod"}
	Group          = P.Field{Group: "Container", Key: "Group"}
	ExposeHostPort = P.Field{Group: "Container", Key: "ExposeHostPort"}
	User           = P.Field{Group: "Container", Key: "User"}
	UserNS         = P.Field{Group: "Container", Key: "UserNS"}
	UIDMap         = P.Field{Group: "Container", Key: "UIDMap", Multiple: true}
	GIDMap         = P.Field{Group: "Container", Key: "GIDMap", Multiple: true}
	SubUIDMap      = P.Field{Group: "Container", Key: "SubUIDMap"}
	SubGIDMap      = P.Field{Group: "Container", Key: "SubGIDMap"}
	RemapUid       = P.Field{Group: "Container", Key: "RemapUid", Multiple: true}
	RemapGid       = P.Field{Group: "Container", Key: "RemapGid", Multiple: true}
	RemapUsers     = P.Field{Group: "Container", Key: "RemapUsers"}
)
