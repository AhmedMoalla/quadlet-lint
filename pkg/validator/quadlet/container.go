package quadlet

import (
	"fmt"
	"strings"

	"github.com/containers/storage/pkg/regexp"

	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

type containerValidator struct {
	ValidatorName string
}

// TODO: - Add reference checks if flag is enabled
// - Add logging to display in debug mode
// - Add source of the rule in the specification
// - Add line and column numbers on the parser's level -> Each entry should have a line associated with it and the starting column of the value
func (c containerValidator) Validate(unit parser.UnitFile) []V.ValidationError {
	return V.DoChecks(c.ValidatorName, unit,
		CheckForAmbiguousImageName(ContainerGroup),
		V.CheckForUnknownKeys(ContainerGroup, supportedContainerKeys),
		V.CheckForUnknownKeys(QuadletGroup, supportedQuadletKeys),
		// One image or rootfs must be specified for the container
		V.CheckForRequiredKey(ContainerGroup, KeyImage, KeyRootfs),
		V.CheckForKeyConflict(ContainerGroup, KeyImage, KeyRootfs),
		// Only allow mixed or control-group, as nothing else works well
		V.CheckForAllowedValues(ServiceGroup, KeyKillMode, "mixed", "control-group"),
		V.CheckForInvalidValuesWithPredicateFn(ContainerGroup, KeyNetwork, func(network string) bool {
			networkName, _, found := strings.Cut(network, ":")
			return found && strings.HasSuffix(networkName, ".container")
		}, "'{value}' is invalid because extra options are not supported when joining another container's network"),
		V.CheckForAllowedValues(ServiceGroup, KeyType, "notify", "oneshot"),
		checkForValidUserAndGroup,
		CheckForUserMappings(ContainerGroup, true),
		V.CheckForInvalidValuesWithMessage(ContainerGroup, KeyExposeHostPort,
			V.MatchesRegex(regexp.Delayed(`\d+(-\d+)?(/udp|/tcp)?$`)).Negate(),
			"'{value}' has invalid port format"),
		V.CheckForInvalidValue(ContainerGroup, KeyPod,
			V.HasLength().And(V.HasSuffix(".pod").Negate())),
	)
}

func checkForValidUserAndGroup(validatorName string, unit parser.UnitFile) *V.ValidationError {
	fmt.Println("checkForValidUserAndGroup:", unit.Filename)
	user, hasUser := unit.Lookup(ContainerGroup, KeyUser)
	okUser := hasUser && len(user) > 0

	group, hasGroup := unit.Lookup(ContainerGroup, KeyGroup)
	okGroup := hasGroup && len(group) > 0

	if !okUser && okGroup {
		return V.Error(validatorName, V.InvalidValue, 0, 0,
			"invalid Group set without User")
	}

	return nil
}
