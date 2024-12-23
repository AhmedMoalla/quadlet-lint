package quadlet

import (
	"fmt"
	"regexp"

	. "github.com/AhmedMoalla/quadlet-lint/pkg/generated/model"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/generated/model/container"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/generated/model/service"
	P "github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/validator/rules"
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

var (
	exposeHostPortRegexp = regexp.MustCompile(`\d+(-\d+)?(/udp|/tcp)?$`)
	networkRegexp        = regexp.MustCompile(`^[^:]+(:(?:\w+=[^,]+,?)+$)*`)
)

// TODO: Implement all these checks in the parser
// TODO: All present values should not be empty
// Check if we have keys that are not listed in the spec
// V.CheckForUnknownKeys(ContainerGroup, supportedContainerKeys),
// V.CheckForUnknownKeys(QuadletGroup, supportedQuadletKeys),
func (c containerValidator) Validate(unit P.UnitFile) []V.ValidationError {
	return CheckRules(c, unit, Groups{
		Container: GContainer{
			Rootfs: Rules(
				RequiredIfNotPresent(Image),
				ConflictsWith(Image),
				CanReference(P.UnitTypeImage, P.UnitTypeBuild),
			),
			Image: Rules(
				ImageNotAmbiguous,
				RequiredIfNotPresent(Rootfs),
				ConflictsWith(Rootfs),
				CanReference(P.UnitTypeImage, P.UnitTypeBuild),
			),
			Network: Rules(
				CanReference(P.UnitTypeNetwork, P.UnitTypeContainer),
				ValuesMust(MatchRegexp(*networkRegexp), Always, "Network value has an invalid format."),
				ValuesMust(HaveFormat(NetworkFormat), Always),
			),
			Volume: Rules(CanReference(P.UnitTypeVolume)),
			Mount:  Rules(CanReference(P.UnitTypeVolume)),
			Pod: Rules(
				HasSuffix(".pod"), // TODO: Add extension as field to UnitType and refer to this as `UnitTypePod.Ext`
				CanReference(P.UnitTypePod),
			),
			Group: Rules(DependsOn(User)),
			RemapUid: Rules(
				Deprecated, ConflictsWithNewUserMappingKeys,
				DependsOn(RemapUsers),
				ValuesMust(HaveZeroOrOneValues, WhenFieldEquals(RemapUsers, "keep-id", "auto"),
					"RemapUsers=keep-id supports only a single value for UID mapping"),
			),
			RemapGid: Rules(
				Deprecated, ConflictsWithNewUserMappingKeys,
				DependsOn(RemapUsers),
				ValuesMust(HaveZeroOrOneValues, WhenFieldEquals(RemapUsers, "keep-id", "auto"),
					"RemapUsers=keep-id supports only a single value for GID mapping"),
			),
			RemapUsers: Rules(
				Deprecated, ConflictsWithNewUserMappingKeys,
				AllowedValues("manual", "auto", "keep-id"),
			),
			ExposeHostPort: Rules(ValuesMust(MatchRegexp(*exposeHostPortRegexp), Always,
				fmt.Sprintf("ExposeHostPort invalid port format. Must match regexp '%s'", exposeHostPortRegexp))),
		},
		Service: GService{
			KillMode: Rules(AllowedValues("mixed", "control-group")),
			Type:     Rules(AllowedValues("notify", "oneshot")),
		},
	})
}
