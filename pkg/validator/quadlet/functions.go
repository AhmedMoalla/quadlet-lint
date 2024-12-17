package quadlet

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

func appendError(errors []validator.ValidationError, err *validator.ValidationError) []validator.ValidationError {
	if err != nil {
		return append(errors, *err)
	}
	return errors
}

func checkForRequiredKey(unit parser.UnitFile, groupName string, requiredKeyCandidates ...string) *validator.ValidationError {
	for _, key := range requiredKeyCandidates {
		if value, _ := unit.Lookup(groupName, key); len(value) > 0 {
			return nil
		}
	}
	return validator.Error(RequiredKey, 0, 0,
		fmt.Sprintf("at least one of these keys is required: %s", requiredKeyCandidates))
}

func checkForKeyConflict(unit parser.UnitFile, groupName string, conflictingKeys ...string) *validator.ValidationError {
	keysFound := make([]string, 0, len(conflictingKeys))
	for _, key := range conflictingKeys {
		if value, _ := unit.Lookup(groupName, key); len(value) > 0 {
			keysFound = append(keysFound, key)
		}
	}

	if len(keysFound) <= 1 {
		return nil
	}

	return validator.Error(KeyConflict, 0, 0,
		fmt.Sprintf("the keys %s cannot be specified together", keysFound))
}

func checkForInvalidValue(unit parser.UnitFile, groupName string, key string, allowedValues ...string) *validator.ValidationError {
	value, ok := unit.Lookup(groupName, key)
	if ok && !slices.Contains(allowedValues, value) {
		return validator.Error(InvalidValue, 0, 0,
			fmt.Sprintf("invalid value '%s' for key '[%s]%s'. Allowed values: %s",
				value, groupName, key, allowedValues))
	}
	return nil
}

func checkForUnknownKeys(unit parser.UnitFile, groupName string, supportedKeys map[string]bool) *validator.ValidationError {
	err := checkForUnknownKeysInSpecificGroup(unit, groupName, supportedKeys)
	if err == nil {
		return checkForUnknownKeysInSpecificGroup(unit, QuadletGroup, supportedQuadletKeys)
	}

	return err
}

func checkForUnknownKeysInSpecificGroup(unit parser.UnitFile, groupName string, supportedKeys map[string]bool) *validator.ValidationError {
	keys := unit.ListKeys(groupName)
	for _, key := range keys {
		if !supportedKeys[key] {
			return validator.Error(UnknownKey, 0, 0,
				fmt.Sprintf("unsupported key '%s' in group '%s' in %s", key, groupName, unit.Path))
		}
	}

	return nil
}

func warnIfAmbiguousName(unit parser.UnitFile, group string) *validator.ValidationError {
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
		return validator.Error(AmbiguousImageName, 0, 0, message)
	}

	return nil
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
