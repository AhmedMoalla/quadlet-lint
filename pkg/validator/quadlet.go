package validator

import (
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	"github.com/containers/podman/v5/pkg/systemd/quadlet"
	"strings"
	"unicode"
)

type QuadletValidator struct {
}

func (q QuadletValidator) Validate(unit parser.UnitFile) []ValidationError {
	validationErrors := make([]ValidationError, 0)

	switch {
	case strings.HasSuffix(unit.Filename, ".container"):
		warnIfAmbiguousName(unit, quadlet.ContainerGroup)

	}

	return validationErrors
}

func warnIfAmbiguousName(unit parser.UnitFile, group string) {
	imageName, ok := unit.Lookup(group, quadlet.KeyImage)
	if !ok {
		return
	}
	if strings.HasSuffix(imageName, ".build") || strings.HasSuffix(imageName, ".image") {
		return
	}
	if !isUnambiguousName(imageName) {
		// TODO: emit error
		// Logf("WarningLevel: %s specifies the image \"%s\" which not a fully qualified image name. This is not ideal for performance and security reasons. See the podman-pull manpage discussion of short-name-aliases.conf for details.", unit.Filename, imageName)
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
