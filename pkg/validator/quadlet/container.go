package quadlet

import (
	"fmt"
	"regexp"

	. "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/container"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/service"
	P "github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/validator/rules"
)

type containerValidator struct {
	name    string
	context V.Context
}

func (v containerValidator) Name() string {
	return v.name
}

func (v containerValidator) Context() V.Context {
	return v.context
}

var (
	exposeHostPortRegexp = regexp.MustCompile(`\d+(-\d+)?(/udp|/tcp)?$`)
	networkRegexp        = regexp.MustCompile(`^[^:]+(:(?:\w+=[^,]+,?)+$)*`)
)

func (v containerValidator) Validate(unit P.UnitFile) []V.ValidationError {
	return CheckRules(v, unit, Groups{
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
				ValuesMust(MatchRegexp(*networkRegexp), Always, "Network value has an invalid format"),
				ValuesMust(HaveFormat(NetworkFormat), Always),
			),
			Volume: Rules(CanReference(P.UnitTypeVolume)),
			Mount:  Rules(CanReference(P.UnitTypeVolume)),
			Pod: Rules(
				HasSuffix(P.UnitTypePod.Ext),
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
