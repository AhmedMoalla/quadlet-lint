package quadlet

import (
	"fmt"
	"strings"
	"unicode"

	M "github.com/AhmedMoalla/quadlet-lint/pkg/model"
	. "github.com/AhmedMoalla/quadlet-lint/pkg/model/generated/container"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
	R "github.com/AhmedMoalla/quadlet-lint/pkg/validator/rules"
)

// ================== Formats ==================

var NetworkFormat = R.Format{
	Name: "Network", ValueSeparator: ":", OptionsSeparator: ",",
	ValidateOptions: func(value string, options map[string]string) error {
		if strings.HasSuffix(value, M.UnitTypeContainer.Ext) && len(options) > 0 {
			return fmt.Errorf("'%s' is invalid because extra options are not supported when "+
				"joining another container's network", value)
		}
		return nil
	},
}

// ================== Rules ==================

var ConflictsWithNewUserMappingKeys = R.ConflictsWith(UserNS, UIDMap, GIDMap, SubUIDMap, SubGIDMap)

func ImageNotAmbiguous(validator V.Validator, unit M.UnitFile, field M.Field) []V.ValidationError {
	if field.Key != Image.Key {
		return nil
	}

	res, found := unit.Lookup(field)
	if !found {
		return nil
	}

	value, ok := res.Value()
	if !ok {
		return R.ErrSlice(validator.Name(), V.InvalidValue, value.Line, value.Column, "value not found")
	}

	imageName := value.Value

	if strings.HasSuffix(imageName, M.UnitTypeBuild.Ext) || strings.HasSuffix(imageName, M.UnitTypeImage.Ext) {
		return nil
	}

	if !isUnambiguousName(imageName) {
		message := fmt.Sprintf("%s specifies the image \"%s\" which not a fully qualified image name. "+
			"This is not ideal for performance and security reasons. "+
			"See the podman-pull manpage discussion of short-name-aliases.conf for details.", unit.FileName(), imageName)
		return R.ErrSlice(validator.Name(), AmbiguousImageName, value.Line, value.Column, message)
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

const hexImageLength = 64

func isImageID(imageName string) bool {
	// All sha25:... names are assumed by podman to be fully specified
	if strings.HasPrefix(imageName, "sha256:") {
		return true
	}

	// However, podman also accepts image ids as pure hex strings,
	// but only those of length 64 are unambiguous image ids
	if len(imageName) != hexImageLength {
		return false
	}

	for _, c := range imageName {
		if !unicode.Is(unicode.Hex_Digit, c) {
			return false
		}
	}

	return true
}
