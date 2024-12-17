package quadlet

import (
	"fmt"
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"github.com/containers/storage/pkg/regexp"
	"strings"
)

type ContainerValidator struct {
}

func (c ContainerValidator) Validate(unit parser.UnitFile) []validator.ValidationError {
	validationErrors := make([]validator.ValidationError, 0)

	validationErrors = appendError(validationErrors, warnIfAmbiguousName(unit, ContainerGroup))
	validationErrors = appendError(validationErrors, checkForUnknownKeys(unit, ContainerGroup, supportedContainerKeys))
	// One image or rootfs must be specified for the container
	validationErrors = appendError(validationErrors, checkForRequiredKey(unit, ContainerGroup, KeyImage, KeyRootfs))
	validationErrors = appendError(validationErrors, checkForKeyConflict(unit, ContainerGroup, KeyImage, KeyRootfs))
	// Only allow mixed or control-group, as nothing else works well
	validationErrors = appendError(validationErrors, checkForInvalidValue(unit, ServiceGroup, KeyKillMode, "mixed", "control-group"))
	validationErrors = appendError(validationErrors, checkIfNetworksValid(unit))
	validationErrors = appendError(validationErrors, checkForInvalidValue(unit, ServiceGroup, KeyType, "notify", "oneshot"))
	validationErrors = appendError(validationErrors, checkIfUserAndGroupValid(unit))
	validationErrors = appendError(validationErrors, checkIfUserMappingsValid(unit, ContainerGroup, true))

	return validationErrors
}

func checkIfPortsValid(unit parser.UnitFile) *validator.ValidationError {
	exposedPorts := unit.LookupAll(ContainerGroup, KeyExposeHostPort)
	for _, exposedPort := range exposedPorts {
		exposedPort = strings.TrimSpace(exposedPort) // Allow whitespace after

		if !isPortRange(exposedPort) {
			return validator.Error(InvalidValue, 0, 0, fmt.Sprintf("invalid port format '%s'", exposedPort))
		}
	}

	// TODO: Validate PublishPort

	return nil
}

var validPortRange = regexp.Delayed(`\d+(-\d+)?(/udp|/tcp)?$`)

func isPortRange(port string) bool {
	return validPortRange.MatchString(port)
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
