package quadlet

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	C "github.com/AhmedMoalla/quadlet-lint/pkg/generated/model/container"
	P "github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	R "github.com/AhmedMoalla/quadlet-lint/pkg/validator/rules"
)

// ================== Formats ==================

var NetworkFormat = R.Format{
	Name: "Network", ValueSeparator: ":", OptionsSeparator: ",",
	ValidateOptions: func(value string, options map[string]string) error {
		if strings.HasSuffix(value, ".container") && len(options) > 0 { // TODO: Add extension as field to UnitType and refer to this as `UnitTypeContainer.Ext`
			return errors.New(fmt.Sprintf("'%s' is invalid because extra options are not supported when "+
				"joining another container's network", value))
		}
		return nil
	},
}

// ================== Rules ==================

var ConflictsWithNewUserMappingKeys = R.ConflictsWith(C.UserNS, C.UIDMap, C.GIDMap, C.SubUIDMap, C.SubGIDMap)

func ImageNotAmbiguous(validator V.Validator, unit P.UnitFile, field P.Field) []V.ValidationError {
	imageName, ok := unit.Lookup(field.Group, C.Image.Key)
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
		return R.ErrSlice(validator.Name(), AmbiguousImageName, 0, 0, message)
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
