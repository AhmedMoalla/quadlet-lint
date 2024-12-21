package quadlet

import (
	"fmt"
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	"slices"
	"strings"
)

var referenceableUnitType = []parser.UnitType{parser.UnitTypeBuild, parser.UnitTypeImage, parser.UnitTypeNetwork,
	parser.UnitTypeContainer, parser.UnitTypePod, parser.UnitTypeVolume}

func CheckForInvalidReference(groupName string, key string) V.CheckerFn {
	return func(validator V.Validator, unit parser.UnitFile) []V.ValidationError {
		fmt.Println("CheckForInvalidReference:", groupName, key)
		if !validator.Context().CheckReferences {
			return nil
		}

		quadletName, _ := unit.Lookup(groupName, key)
		return checkIfQuadletReferenceExists(validator, quadletName)
	}
}

func CheckForInvalidReferences(groupName string, key string) V.CheckerFn {
	return func(validator V.Validator, unit parser.UnitFile) []V.ValidationError {
		fmt.Println("CheckForInvalidReferences:", groupName, key)
		context := validator.Context()
		if !context.CheckReferences {
			return nil
		}

		validationErrors := make([]V.ValidationError, 0)
		quadletNames := unit.LookupAll(groupName, key)
		for _, quadletName := range quadletNames {
			validationErrors = append(validationErrors, checkIfQuadletReferenceExists(validator, quadletName)...)
		}
		return validationErrors
	}
}

func checkIfQuadletReferenceExists(validator V.Validator, quadletName string) []V.ValidationError {
	units := validator.Context().AllUnitFiles
	quadletName, _, _ = strings.Cut(quadletName, ":") // Sometimes values have options after ':' like networks
	for _, unitType := range referenceableUnitType {
		if strings.HasSuffix(quadletName, string("."+unitType)) {
			foundUnit := slices.ContainsFunc(units, func(unit parser.UnitFile) bool {
				return unit.Filename == quadletName
			})

			if !foundUnit {
				return V.ErrSlice(validator.Name(), InvalidReference, 0, 0,
					fmt.Sprintf("requested Quadlet %s '%s' was not found", unitType, quadletName))
			}
		}
	}
	return nil
}

func CheckForUserMappings(groupName string, supportManual bool) V.CheckerFn {
	return func(validator V.Validator, unit parser.UnitFile) []V.ValidationError {
		fmt.Println("CheckForUserMappings:", unit.Filename, groupName)
		if mappingsDefined(unit, groupName) {
			_, hasRemapUID := unit.Lookup(groupName, KeyRemapUid)
			_, hasRemapGID := unit.Lookup(groupName, KeyRemapGid)
			_, RemapUsers := unit.LookupLast(groupName, KeyRemapUsers)
			if hasRemapUID || hasRemapGID || RemapUsers {
				return V.ErrSlice(validator.Name(), V.DeprecatedKey, 0, 0,
					"deprecated Remap keys are set along with explicit mapping keys")
			}
			return nil
		}

		return checkIfUserRemapsValid(validator, unit, groupName, supportManual)
	}
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

func checkIfUserRemapsValid(validator V.Validator, unitFile parser.UnitFile, groupName string, supportManual bool) []V.ValidationError {
	uidMaps := unitFile.LookupAllStrv(groupName, KeyRemapUid)
	gidMaps := unitFile.LookupAllStrv(groupName, KeyRemapGid)
	remapUsers, _ := unitFile.LookupLast(groupName, KeyRemapUsers)
	switch remapUsers {
	case "":
		if len(uidMaps) > 0 {
			return V.ErrSlice(validator.Name(), V.RequiredKey, 0, 0,
				"UidMap set without RemapUsers")
		}
		if len(gidMaps) > 0 {
			return V.ErrSlice(validator.Name(), V.RequiredKey, 0, 0,
				"GidMap set without RemapUsers")
		}
	case "manual":
		if !supportManual {
			return V.ErrSlice(validator.Name(), V.InvalidValue, 0, 0,
				"RemapUsers=manual is not supported")
		}
	case "auto":
	case "keep-id":
		if len(uidMaps) > 0 {
			if len(uidMaps) > 1 {
				return V.ErrSlice(validator.Name(), V.InvalidValue, 0, 0,
					"RemapUsers=keep-id supports only a single value for UID mapping")
			}
		}
		if len(gidMaps) > 0 {
			if len(gidMaps) > 1 {
				return V.ErrSlice(validator.Name(), V.InvalidValue, 0, 0,
					"RemapUsers=keep-id supports only a single value for GID mapping")
			}
		}
	default:
		return V.ErrSlice(validator.Name(), V.InvalidValue, 0, 0,
			fmt.Sprintf("unsupported RemapUsers option '%s'", remapUsers))
	}

	return nil
}

func CheckForAmbiguousImageName(group string) V.CheckerFn {
	return func(validator V.Validator, unit parser.UnitFile) []V.ValidationError {
		fmt.Println("CheckForAmbiguousImageName:", unit.Filename, group)
		imageName, ok := unit.Lookup(group, KeyImage)
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
			return V.ErrSlice(validator.Name(), AmbiguousImageName, 0, 0, message)
		}

		return nil
	}
}
