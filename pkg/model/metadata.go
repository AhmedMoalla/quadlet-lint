//go:generate go run ../../cmd/quadlet-model-gen --podman-version v5.3.1
package model

// AdditionalFields is a mapping from Group to Fields.
// This is used by the generator to define additional fields on service struct that were not defined as variables
// in Podman because they were used as string literals
// TODO: Generate with the generator by checking the parameters to Lookup functions
var AdditionalFields = map[string][]string{
	"Service": {"KillMode", "Type"},
}
