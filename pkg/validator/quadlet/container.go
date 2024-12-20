package quadlet

import (
	"fmt"
	"strings"

	"github.com/containers/storage/pkg/regexp"

	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

type containerValidator struct {
	name    string
	context V.Context
}

func (c containerValidator) Name() string {
	return c.name
}

func (c containerValidator) Context() V.Context {
	return c.context
}

// TODO:
// - Add logging to display in debug mode
// - Add source of the rule in the specification
// - Add line and column numbers on the parser's level -> Each entry should have a line associated with it and the starting column of the value
func (c containerValidator) Validate(unit parser.UnitFile) []V.ValidationError {
	return V.DoChecks(c, unit,
		CheckForAmbiguousImageName(ContainerGroup),

		// Check if we have keys that are not listed in the spec
		V.CheckForUnknownKeys(ContainerGroup, supportedContainerKeys),
		V.CheckForUnknownKeys(QuadletGroup, supportedQuadletKeys),

		// One image or rootfs must be specified for the container
		V.CheckForRequiredKey(ContainerGroup, KeyImage, KeyRootfs),
		V.CheckForKeyConflict(ContainerGroup, KeyImage, KeyRootfs),

		// Check if image refers to an existing .image or .build quadlet
		CheckForInvalidReference(ContainerGroup, KeyImage),
		CheckForInvalidReference(ContainerGroup, KeyRootfs),

		// Only allow mixed or control-group, as nothing else works well
		V.CheckForAllowedValues(ServiceGroup, KeyKillMode, "mixed", "control-group"),

		// When referring to a .container quadlet options are not supported
		V.CheckForInvalidValuesWithPredicateFn(ContainerGroup, KeyNetwork, func(network string) bool {
			networkName, _, found := strings.Cut(network, ":")
			return found && strings.HasSuffix(networkName, ".container")
		}, "'{value}' is invalid because extra options are not supported when joining another container's network"),

		// Check if network refers to an existing .network or .container
		CheckForInvalidReferences(ContainerGroup, KeyNetwork),

		V.CheckForAllowedValues(ServiceGroup, KeyType, "notify", "oneshot"),

		checkForValidUserAndGroup,
		CheckForUserMappings(ContainerGroup, true),

		V.CheckForInvalidValuesWithMessage(ContainerGroup, KeyExposeHostPort,
			V.MatchesRegex(regexp.Delayed(`\d+(-\d+)?(/udp|/tcp)?$`)).Negate(),
			"'{value}' has invalid port format"),

		// Check if pod refers to an existing .pod quadlet
		V.CheckForInvalidValue(ContainerGroup, KeyPod,
			V.HasLength().And(V.HasSuffix(".pod").Negate())),
		CheckForInvalidReference(ContainerGroup, KeyPod),

		// Check if volume refers to an existing .volume quadlet
		CheckForInvalidReferences(ContainerGroup, KeyVolume),
		CheckForInvalidReferences(ContainerGroup, KeyMount),

		// TODO: Check for Mount see: pkg/systemd/quadlet/quadlet.go:2064
	)
}

func checkForValidUserAndGroup(validator V.Validator, unit parser.UnitFile) []V.ValidationError {
	fmt.Println("checkForValidUserAndGroup:", unit.Filename)
	user, hasUser := unit.Lookup(ContainerGroup, KeyUser)
	okUser := hasUser && len(user) > 0

	group, hasGroup := unit.Lookup(ContainerGroup, KeyGroup)
	okGroup := hasGroup && len(group) > 0

	if !okUser && okGroup {
		return V.ErrorAsSlice(validator.Name(), V.InvalidValue, 0, 0, "invalid Group set without User")
	}

	return nil
}
