package quadlet

import (
	"fmt"
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/containers/podman/v5/pkg/systemd/quadlet"
	"github.com/containers/storage/pkg/regexp"
	"strings"
	"unicode"
)

type containerValidator struct {
	ValidatorName string
}

func (c containerValidator) Validate(unit parser.UnitFile) []validator.ValidationError {
	return validator.DoChecks(c.ValidatorName, unit,
		CheckForAmbiguousImageName(ContainerGroup),
		validator.CheckForUnknownKeys(ContainerGroup, supportedContainerKeys),
		validator.CheckForUnknownKeys(QuadletGroup, supportedQuadletKeys),
		// One image or rootfs must be specified for the container
		validator.CheckForRequiredKey(ContainerGroup, KeyImage, KeyRootfs),
		validator.CheckForKeyConflict(ContainerGroup, KeyImage, KeyRootfs),
		// Only allow mixed or control-group, as nothing else works well
		validator.CheckForAllowedValues(ServiceGroup, KeyKillMode, "mixed", "control-group"),
		checkIfNetworksValid(unit),
		validator.CheckForAllowedValues(ServiceGroup, KeyType, "notify", "oneshot"),
		checkIfUserAndGroupValid(unit),
		checkIfUserMappingsValid(unit, ContainerGroup, true),
		validator.CheckForInvalidValuesWithMessage(ContainerGroup, KeyExposeHostPort,
			validator.MatchesRegex(regexp.Delayed(`\d+(-\d+)?(/udp|/tcp)?$`)).Negate(),
			"'{value}' has invalid port format"),
		validator.CheckForInvalidValue(ContainerGroup, KeyPod, validator.HasLength().And(validator.HasSuffix(".pod").Negate())),
	)
}

func CheckForAmbiguousImageName(group string) validator.CheckerFn {
	return func(validatorName string, unit parser.UnitFile) *validator.ValidationError {
		imageName, ok := unit.Lookup(group, quadlet.KeyImage)
		if !ok {
			return nil
		}

		if strings.HasSuffix(imageName, ".build") || strings.HasSuffix(imageName, ".image") {
			return nil
		}

		if !isUnambiguousName(imageName) {
			message := fmt.Sprintf("%s specifies the image \"%s\" which not a fully qualified image name. "+
				"This is not ideal for performance and security reasons. "+
				"See the podman-pull manpage discussion of short-name-aliases.conf for details.", unit.Filename, imageName)
			return validator.Error(validatorName, AmbiguousImageName, 0, 0, message)
		}

		return nil
	}
}

func isUnambiguousName(imageName string) bool {
	// Fully specified image ids are unambiguous
	if isImageID(imageName) {
		return true
	}

	// Otherwise we require a fully qualified name
	firstSlash := strings.Index(imageName, "/")
	if firstSlash == -1 {
		// No domain or path, not fully qualified
		return false
	}

	// What is before the first slash can be a domain or a path
	domain := imageName[:firstSlash]

	// If its a domain (has dot or port or is "localhost") it is considered fq
	if strings.ContainsAny(domain, ".:") || domain == "localhost" {
		return true
	}

	return false
}

func isImageID(imageName string) bool {
	// All sha25:... names are assumed by podman to be fully specified
	if strings.HasPrefix(imageName, "sha256:") {
		return true
	}

	// However, podman also accepts image ids as pure hex strings,
	// but only those of length 64 are unambiguous image ids
	if len(imageName) != 64 {
		return false
	}

	for _, c := range imageName {
		if !unicode.Is(unicode.Hex_Digit, c) {
			return false
		}
	}

	return true
}

func checkIfUserMappingsValid(unit parser.UnitFile, groupName string, supportManual bool) *validator.ValidationError {
	if mappingsDefined(unit, groupName) {
		_, hasRemapUID := unit.Lookup(groupName, KeyRemapUid)
		_, hasRemapGID := unit.Lookup(groupName, KeyRemapGid)
		_, RemapUsers := unit.LookupLast(groupName, KeyRemapUsers)
		if hasRemapUID || hasRemapGID || RemapUsers {
			return validator.Error(DeprecatedKey, 0, 0,
				"deprecated Remap keys are set along with explicit mapping keys")
		}
		return nil
	}

	return checkIfUserRemapsValid(unit, groupName, supportManual)
}

func mappingsDefined(unit parser.UnitFile, groupName string) bool {
	if userns, ok := unit.Lookup(groupName, KeyUserNS); ok && len(userns) > 0 {
		return true
	}

	if len(unit.LookupAllStrv(groupName, KeyUIDMap)) > 0 {
		return true
	}

	if len(unit.LookupAllStrv(groupName, KeyGIDMap)) > 0 {
		return true
	}

	if subUIDMap, ok := unit.Lookup(groupName, KeySubUIDMap); ok && len(subUIDMap) > 0 {
		return true
	}

	if subGIDMap, ok := unit.Lookup(groupName, KeySubGIDMap); ok && len(subGIDMap) > 0 {
		return true
	}

	return false
}

func checkIfUserRemapsValid(unitFile parser.UnitFile, groupName string, supportManual bool) *validator.ValidationError {
	uidMaps := unitFile.LookupAllStrv(groupName, KeyRemapUid)
	gidMaps := unitFile.LookupAllStrv(groupName, KeyRemapGid)
	remapUsers, _ := unitFile.LookupLast(groupName, KeyRemapUsers)
	switch remapUsers {
	case "":
		if len(uidMaps) > 0 {
			return validator.Error(RequiredKey, 0, 0, "UidMap set without RemapUsers")
		}
		if len(gidMaps) > 0 {
			return validator.Error(RequiredKey, 0, 0, "GidMap set without RemapUsers")
		}
	case "manual":
		if !supportManual {
			return validator.Error(InvalidValue, 0, 0, "RemapUsers=manual is not supported")
		}
	case "auto":
	case "keep-id":
		if len(uidMaps) > 0 {
			if len(uidMaps) > 1 {
				return validator.Error(InvalidValue, 0, 0, "RemapUsers=keep-id supports only a single value for UID mapping")
			}
		}
		if len(gidMaps) > 0 {
			if len(gidMaps) > 1 {
				return validator.Error(InvalidValue, 0, 0, "RemapUsers=keep-id supports only a single value for GID mapping")
			}
		}
	default:
		return validator.Error(InvalidValue, 0, 0,
			fmt.Sprintf("unsupported RemapUsers option '%s'", remapUsers))
	}

	return nil
}

func checkIfUserAndGroupValid(unit parser.UnitFile) *validator.ValidationError {
	user, hasUser := unit.Lookup(ContainerGroup, KeyUser)
	okUser := hasUser && len(user) > 0

	group, hasGroup := unit.Lookup(ContainerGroup, KeyGroup)
	okGroup := hasGroup && len(group) > 0

	if !okUser && okGroup {
		return validator.Error(InvalidValue, 0, 0, "invalid Group set without User")
	}

	return nil
}

func checkIfNetworksValid(unit parser.UnitFile) *validator.ValidationError {
	networks := unit.LookupAll(ContainerGroup, KeyNetwork)
	for _, network := range networks {
		if len(network) == 0 {
			continue
		}

		networkName, _, found := strings.Cut(network, ":")
		if found && strings.HasSuffix(networkName, ".container") {
			return validator.Error(InvalidValue, 0, 0,
				"extra options are not supported when joining another container's network")
		}
	}
	return nil
}
