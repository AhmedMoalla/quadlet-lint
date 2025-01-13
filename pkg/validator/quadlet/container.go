package quadlet

import (
	"regexp"

	M "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/container"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/service"
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

func (v containerValidator) Validate(unit M.UnitFile) []V.ValidationError {
	return CheckRules(v, unit, Groups{
		Container: GContainer{
			Rootfs: Rules(
				RequiredIfNotPresent(Image),
				ConflictsWith(Image),
				CanReference(M.UnitTypeImage, M.UnitTypeBuild),
			),
			Image: Rules(
				ImageNotAmbiguous,
				RequiredIfNotPresent(Rootfs),
				ConflictsWith(Rootfs),
				CanReference(M.UnitTypeImage, M.UnitTypeBuild),
			),
			Network: Rules(
				CanReference(M.UnitTypeNetwork, M.UnitTypeContainer),
				MatchRegexp(networkRegexp),
				HaveFormat(NetworkFormat),
			),
			Volume: Rules(CanReference(M.UnitTypeVolume)),
			Mount:  Rules(CanReference(M.UnitTypeVolume)),
			Pod: Rules(
				HasSuffix(M.UnitTypePod.Ext),
				CanReference(M.UnitTypePod),
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
			ExposeHostPort: Rules(MatchRegexp(exposeHostPortRegexp)),
		},
		Service: GService{
			KillMode: Rules(AllowedValues("mixed", "control-group")),
			Type:     Rules(AllowedValues("notify", "oneshot")),
		},
	})
}
