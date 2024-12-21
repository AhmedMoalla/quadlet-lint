package model

import (
	P "github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator/generated/model/container"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator/generated/model/service"
)

type Groups struct {
	Container container.Container
	Service   service.Service
}

var Fields = map[string]P.Field{
	// Container group fields
	"Image":          container.Image,
	"Rootfs":         container.Rootfs,
	"Network":        container.Network,
	"Volume":         container.Volume,
	"Mount":          container.Mount,
	"Pod":            container.Pod,
	"Group":          container.Group,
	"ExposeHostPort": container.ExposeHostPort,
	"User":           container.User,
	"UserNS":         container.UserNS,
	"UIDMap":         container.UIDMap,
	"GIDMap":         container.GIDMap,
	"SubUIDMap":      container.SubUIDMap,
	"SubGIDMap":      container.SubGIDMap,
	"RemapUid":       container.RemapUid,
	"RemapGid":       container.RemapGid,
	"RemapUsers":     container.RemapUsers,

	// Service group fields
	"KillMode": service.KillMode,
	"Type":     service.Type,
}
