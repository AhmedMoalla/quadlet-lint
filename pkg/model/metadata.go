//go:generate go run ../../cmd/quadlet-model-gen --podman-version v5.3.1
package model

// AdditionalFields is a mapping from Group to Fields.
// This is used by the generator to define additional fields on service struct that were not defined as variables
// in Podman because they were used as string literals
var AdditionalFields = map[string][]string{
	"Service": {"KillMode", "Type"},
}

type FieldMetadata struct {
	Multiple bool
}

// FieldsMetadata gives additional metadata that is not present in Podman's source code.
// This is used by the generator to add fields to the generated Field object map.
// TODO: Generate with the generator by finding calls to parser Lookup functions
var FieldsMetadata = map[string]map[string]FieldMetadata{
	"Container": {
		"AddDevice":            FieldMetadata{Multiple: true},
		"Annotation":           FieldMetadata{Multiple: true},
		"ContainersConfModule": FieldMetadata{Multiple: true},
		"DNS":                  FieldMetadata{Multiple: true},
		"DNSOption":            FieldMetadata{Multiple: true},
		"DNSSearch":            FieldMetadata{Multiple: true},
		"ExposeHostPort":       FieldMetadata{Multiple: true},
		"GIDMap":               FieldMetadata{Multiple: true},
		"GlobalArgs":           FieldMetadata{Multiple: true},
		"Label":                FieldMetadata{Multiple: true},
		"Network":              FieldMetadata{Multiple: true},
		"NetworkAlias":         FieldMetadata{Multiple: true},
		"PodmanArgs":           FieldMetadata{Multiple: true},
		"PublishPort":          FieldMetadata{Multiple: true},
		"Tmpfs":                FieldMetadata{Multiple: true},
		"UIDMap":               FieldMetadata{Multiple: true},
		"Ulimit":               FieldMetadata{Multiple: true},
		"Volume":               FieldMetadata{Multiple: true},
	},
	"Pod": {
		"ContainersConfModule": FieldMetadata{Multiple: true},
		"DNS":                  FieldMetadata{Multiple: true},
		"DNSOption":            FieldMetadata{Multiple: true},
		"DNSSearch":            FieldMetadata{Multiple: true},
		"GIDMap":               FieldMetadata{Multiple: true},
		"GlobalArgs":           FieldMetadata{Multiple: true},
		"Network":              FieldMetadata{Multiple: true},
		"NetworkAlias":         FieldMetadata{Multiple: true},
		"PodmanArgs":           FieldMetadata{Multiple: true},
		"PublishPort":          FieldMetadata{Multiple: true},
		"UIDMap":               FieldMetadata{Multiple: true},
		"Volume":               FieldMetadata{Multiple: true},
	},
	"Kube": {
		"ContainersConfModule": FieldMetadata{Multiple: true},
		"GlobalArgs":           FieldMetadata{Multiple: true},
		"Network":              FieldMetadata{Multiple: true},
		"PodmanArgs":           FieldMetadata{Multiple: true},
		"PublishPort":          FieldMetadata{Multiple: true},
	},
	"Network": {
		"ContainersConfModule": FieldMetadata{Multiple: true},
		"DNS":                  FieldMetadata{Multiple: true},
		"Gateway":              FieldMetadata{Multiple: true},
		"GlobalArgs":           FieldMetadata{Multiple: true},
		"IPRange":              FieldMetadata{Multiple: true},
		"Label":                FieldMetadata{Multiple: true},
		"PodmanArgs":           FieldMetadata{Multiple: true},
		"Subnet":               FieldMetadata{Multiple: true},
	},
	"Volume": {
		"ContainersConfModule": FieldMetadata{Multiple: true},
		"GlobalArgs":           FieldMetadata{Multiple: true},
		"Label":                FieldMetadata{Multiple: true},
		"PodmanArgs":           FieldMetadata{Multiple: true},
	},
	"Build": {
		"ContainersConfModule": FieldMetadata{Multiple: true},
		"GlobalArgs":           FieldMetadata{Multiple: true},
		"Network":              FieldMetadata{Multiple: true},
		"PodmanArgs":           FieldMetadata{Multiple: true},
		"Volume":               FieldMetadata{Multiple: true},
	},
	"Image": {
		"ContainersConfModule": FieldMetadata{Multiple: true},
		"GlobalArgs":           FieldMetadata{Multiple: true},
		"PodmanArgs":           FieldMetadata{Multiple: true},
	},
}
