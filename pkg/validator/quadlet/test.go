package quadlet

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/containers/storage/pkg/regexp"

	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

type NewValidator struct {
	VName    string
	VContext V.Context
}

func (n NewValidator) Name() string {
	return n.VName
}

func (n NewValidator) Context() V.Context {
	return n.VContext
}

// TODO: Refactor
func CheckRules(validator V.Validator, unit parser.UnitFile, rules Groups) []V.ValidationError {
	validationErrors := make([]V.ValidationError, 0)

	groupsValue := reflect.ValueOf(rules)
	groupsType := reflect.TypeOf(rules)

	for groupIndex := range groupsType.NumField() {
		groupField := groupsType.Field(groupIndex)
		groupValue := groupsValue.Field(groupIndex)

		groupType := groupField.Type
		for fieldIndex := range groupType.NumField() {
			fieldType := groupType.Field(fieldIndex)

			fieldName := fieldType.Name

			ruleFns := groupValue.FieldByName(fieldName).Interface().([]Rule)
			for _, rule := range ruleFns {
				field, ok := Fields[fieldName]
				if !ok {
					panic(fmt.Sprintf("field %s not found in Fields map", fieldName))
				}
				validationErrors = append(validationErrors, rule(validator, unit, field)...)
			}
		}
	}

	return validationErrors
}

// Format is the format that a value of a given key has
// For example: the Network key has the format: mode[:options,...] so the Format would be:
// - ValueSeparator: ":"
// - OptionsSeparator: ","
type Format struct {
	Name             string // Name of the Format
	ValueSeparator   string // ValueSeparator is the separator between the value and its options
	OptionsSeparator string // OptionsSeparator is the separator between the options

	Value string // Value is the value before the ValueSeparator. Populated after calling ParseAndValidate.
	// Options are the options after the ValueSeparator split by the OptionsSeparator.
	// Populated after calling ParseAndValidate.
	Options map[string]string

	ValidateOptions func(value string, options map[string]string) error
}

func (f *Format) ParseAndValidate(value string) error {
	split := strings.Split(value, f.ValueSeparator)
	if len(split) == 0 || len(split) > 2 {
		return errors.New(
			fmt.Sprintf("'%s' does not match the '%s' format because it is expected to have 2 parts after "+
				"splitting the value with '%s' but got instead %d parts", value, f.Name, f.ValueSeparator, len(split)))
	}

	f.Value = split[0]

	if len(split) == 1 { // no options
		return nil
	}

	split = strings.Split(split[1], f.OptionsSeparator)
	options := make(map[string]string, len(split))
	for _, pair := range split {
		kv := strings.Split(pair, "=")
		options[kv[0]] = kv[1]
	}
	f.Options = options

	if f.ValidateOptions != nil {
		if err := f.ValidateOptions(f.Value, f.Options); err != nil {
			return err
		}
	}

	return nil
}

var NetworkFormat = Format{
	Name: "Network", ValueSeparator: ":", OptionsSeparator: ",",
	ValidateOptions: func(value string, options map[string]string) error {
		if strings.HasSuffix(value, ".container") && len(options) > 0 { // TODO: Add extension as field to UnitType and refer to this as `UnitTypeContainer.Ext`
			return errors.New(fmt.Sprintf("'%s' is invalid because extra options are not supported when "+
				"joining another container's network", value))
		}
		return nil
	},
}

func HaveFormat(format Format) ValuesValidator {
	return func(validator V.Validator, field Field, values []string) *V.ValidationError {
		for _, value := range values {
			err := format.ParseAndValidate(value)
			if err != nil {
				return V.Err(validator.Name(), V.InvalidValue, 0, 0, err.Error())
			}
		}

		return nil
	}
}

// TODO: Implement all these checks in the parser
// TODO: All present values should not be empty
// Check if we have keys that are not listed in the spec
// V.CheckForUnknownKeys(ContainerGroup, supportedContainerKeys),
// V.CheckForUnknownKeys(QuadletGroup, supportedQuadletKeys),
func (n NewValidator) Validate(unit parser.UnitFile) []V.ValidationError {
	return CheckRules(n, unit, Groups{
		Container: Container{
			Rootfs: Rules(
				RequiredIfNotPresent(Image),
				ConflictsWith(Image),
				CanReference(parser.UnitTypeImage, parser.UnitTypeBuild),
			),
			Image: Rules(
				ImageNotAmbiguous,
				RequiredIfNotPresent(Rootfs),
				ConflictsWith(Rootfs),
				CanReference(parser.UnitTypeImage, parser.UnitTypeBuild),
			),
			Network: Rules(
				CanReference(parser.UnitTypeNetwork, parser.UnitTypeContainer),
				ValuesMust(MatchRegexp(networkRegexp), Always, "Network value has an invalid format."),
				ValuesMust(HaveFormat(NetworkFormat), Always),
			),
			Volume: Rules(CanReference(parser.UnitTypeVolume)),
			Mount:  Rules(CanReference(parser.UnitTypeVolume)),
			Pod: Rules(
				HasSuffix(".pod"), // TODO: Add extension as field to UnitType and refer to this as `UnitTypePod.Ext`
				CanReference(parser.UnitTypePod),
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
			ExposeHostPort: Rules(ValuesMust(MatchRegexp(exposeHostPortRegexp), Always,
				fmt.Sprintf("ExposeHostPort invalid port format. Must match regexp '%s'", exposeHostPortRegexp))),
		},
		Service: Service{
			KillMode: Rules(AllowedValues("mixed", "control-group")),
			Type:     Rules(AllowedValues("notify", "oneshot")),
		},
	})
}

func Always(V.Validator, parser.UnitFile, Field) bool {
	return true
}

func MatchRegexp(regexp regexp.Regexp) ValuesValidator {
	return func(validator V.Validator, field Field, values []string) *V.ValidationError {
		for _, value := range values {
			if !regexp.MatchString(value) {
				return V.Err(validator.Name(), V.InvalidValue, 0, 0,
					fmt.Sprintf("Must match regexp '%s'", regexp))
			}
		}
		return nil
	}
}

type Groups struct {
	Container Container
	Service   Service
}

type Container struct {
	Rootfs         []Rule
	Image          []Rule
	Network        []Rule
	Volume         []Rule
	Mount          []Rule
	Pod            []Rule
	Group          []Rule
	RemapUid       []Rule
	RemapGid       []Rule
	RemapUsers     []Rule
	ExposeHostPort []Rule
}

type Service struct {
	KillMode []Rule
	Type     []Rule
}

type Field struct {
	Group    string
	Key      string
	Multiple bool
}

func (f Field) String() string {
	return fmt.Sprintf("%s.%s", f.Group, f.Key)
}

var (
	// Container Group fields
	Image          = Field{Group: "Container", Key: "Image"}
	Rootfs         = Field{Group: "Container", Key: "Rootfs"}
	Network        = Field{Group: "Container", Key: "Network", Multiple: true}
	Volume         = Field{Group: "Container", Key: "Volume"}
	Mount          = Field{Group: "Container", Key: "Mount"}
	Pod            = Field{Group: "Container", Key: "Pod"}
	Group          = Field{Group: "Container", Key: "Group"}
	ExposeHostPort = Field{Group: "Container", Key: "ExposeHostPort"}
	User           = Field{Group: "Container", Key: "User"}
	UserNS         = Field{Group: "Container", Key: "UserNS"}
	UIDMap         = Field{Group: "Container", Key: "UIDMap", Multiple: true}
	GIDMap         = Field{Group: "Container", Key: "GIDMap", Multiple: true}
	SubUIDMap      = Field{Group: "Container", Key: "SubUIDMap"}
	SubGIDMap      = Field{Group: "Container", Key: "SubGIDMap"}
	RemapUid       = Field{Group: "Container", Key: "RemapUid", Multiple: true}
	RemapGid       = Field{Group: "Container", Key: "RemapGid", Multiple: true}
	RemapUsers     = Field{Group: "Container", Key: "RemapUsers"}

	// Service Group fields
	KillMode = Field{Group: "Container", Key: "KillMode"}
	Type     = Field{Group: "Container", Key: "Type"}

	Fields = map[string]Field{
		"Image":          Image,
		"Rootfs":         Rootfs,
		"Network":        Network,
		"Volume":         Volume,
		"Mount":          Mount,
		"Pod":            Pod,
		"Group":          Group,
		"ExposeHostPort": ExposeHostPort,
		"User":           User,
		"UserNS":         UserNS,
		"UIDMap":         UIDMap,
		"GIDMap":         GIDMap,
		"SubUIDMap":      SubUIDMap,
		"SubGIDMap":      SubGIDMap,
		"RemapUid":       RemapUid,
		"RemapGid":       RemapGid,
		"RemapUsers":     RemapUsers,
		"KillMode":       KillMode,
		"Type":           Type,
	}
)

type Rule = func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError

func Rules(rules ...Rule) []Rule {
	return rules
}

func RequiredIfNotPresent(other Field) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
		if !unit.HasValue(other.Group, other.Key) && !unit.HasValue(field.Group, field.Key) {
			return V.ErrSlice(validator.Name(), V.RequiredKey, 0, 0,
				fmt.Sprintf("at least one of these keys is required: %s, %s", field, other))
		}

		return nil
	}
}

func ConflictsWith(others ...Field) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
		validationErrors := make([]V.ValidationError, 0)
		for _, other := range others {
			if unit.HasValue(other.Group, other.Key) && unit.HasValue(field.Group, field.Key) {
				validationErrors = append(validationErrors, *V.Err(validator.Name(), V.KeyConflict, 0, 0,
					fmt.Sprintf("the keys %s, %s cannot be specified together", field, other)))
			}
		}

		return validationErrors
	}
}

func CanReference(unitTypes ...parser.UnitType) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
		context := validator.Context()
		if !context.CheckReferences {
			return nil
		}

		var values []string
		if field.Multiple {
			values = unit.LookupAll(field.Group, field.Key)
		} else if value, found := unit.Lookup(field.Group, field.Key); found {
			values = []string{value}
		}

		if len(values) == 0 {
			return nil
		}

		units := context.AllUnitFiles
		validationErrors := make([]V.ValidationError, 0)
		for _, value := range values {
			for _, unitType := range unitTypes {
				if strings.HasSuffix(value, string("."+unitType)) { // TODO: Add extension as field to UnitType
					foundUnit := slices.ContainsFunc(units, func(unit parser.UnitFile) bool {
						return unit.Filename == value
					})

					if !foundUnit {
						validationErrors = append(validationErrors, *V.Err(validator.Name(), InvalidReference, 0, 0,
							fmt.Sprintf("requested Quadlet %s '%s' was not found", unitType, value)))
					}
				}

				break
			}
		}

		return validationErrors
	}
}

func ImageNotAmbiguous(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
	imageName, ok := unit.Lookup(field.Group, Image.Key)
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

func AllowedValues(allowedValues ...string) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
		value, ok := unit.Lookup(field.Group, field.Key)
		if ok && !slices.Contains(allowedValues, value) {
			return V.ErrSlice(validator.Name(), V.InvalidValue, 0, 0,
				fmt.Sprintf("invalid value '%s' for key '%s'. Allowed values: %s",
					value, field, allowedValues))
		}
		return nil
	}
}

func HasSuffix(suffix string) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
		value, found := unit.Lookup(field.Group, field.Key)
		if !found {
			return nil
		}

		if !strings.HasSuffix(value, suffix) {
			return V.ErrSlice(validator.Name(), V.InvalidValue, 0, 0,
				fmt.Sprintf("value '%s' must have suffix '%s'", value, suffix))
		}

		return nil
	}
}

func DependsOn(dependency Field) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
		dependency, dependencyFound := unit.Lookup(dependency.Group, dependency.Key)
		dependencyOk := dependencyFound && len(dependency) > 0

		value, found := unit.Lookup(field.Group, field.Key)
		fieldOk := found && len(value) > 0

		if !dependencyOk && fieldOk {
			return V.ErrSlice(validator.Name(), V.UnsatisfiedDependency, 0, 0,
				fmt.Sprintf("value for '%s' was set but it depends on key '%s' which was not found",
					field, dependency))
		}

		return nil
	}
}

func Deprecated(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
	if _, found := unit.Lookup(field.Group, field.Key); found {
		return V.ErrSlice(validator.Name(), V.DeprecatedKey, 0, 0,
			fmt.Sprintf("key '%s' is deprecated and should not be used", field))
	}

	return nil
}

func ValueDependsOn(dependency Field, conditionValue string, valueValidator func(values []string) bool, message string) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
		dependencyValue, found := unit.Lookup(dependency.Group, dependency.Key)
		values := unit.LookupAll(field.Group, field.Key)
		if found && dependencyValue == conditionValue && !valueValidator(values) {
			return V.ErrSlice(validator.Name(), V.InvalidValue, 0, 0,
				fmt.Sprintf("value '%s' of field '%s' is invalid when %s=%s because %s",
					values, field, dependency, conditionValue, message))
		}

		return nil
	}
}

var ConflictsWithNewUserMappingKeys = ConflictsWith(UserNS, UIDMap, GIDMap, SubUIDMap, SubGIDMap)

func HaveZeroOrOneValues(validator V.Validator, field Field, values []string) *V.ValidationError {
	if len(values) > 1 {
		return V.Err(validator.Name(), V.InvalidValue, 0, 0, "should have exactly zero or one value")
	}

	return nil
}

type ValuesValidator func(validator V.Validator, field Field, values []string) *V.ValidationError
type RulePredicate func(validator V.Validator, unit parser.UnitFile, field Field) bool

func WhenFieldEquals(conditionField Field, conditionValues ...string) RulePredicate {
	return func(validator V.Validator, unit parser.UnitFile, field Field) bool {
		values := unit.LookupAll(conditionField.Group, conditionField.Key)
		for _, fieldValue := range values {
			for _, conditionValue := range conditionValues {
				if fieldValue == conditionValue {
					return true
				}
			}
		}
		return false
	}
}

func ValuesMust(valuePredicate ValuesValidator, rulePredicate RulePredicate, messageAndArgs ...any) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
		if rulePredicate(validator, unit, field) {
			// TODO: Should use correct Lookup function depending on the field
			// Refactor Lookup function to take Field instances
			// Fields should define LookupMode property that tells which Lookup function to use
			values := unit.LookupAllStrv(field.Group, field.Key)
			if err := valuePredicate(validator, field, values); err != nil {
				errorMsg := buildErrorMessage(messageAndArgs, err)
				return V.ErrSlice(validator.Name(), V.InvalidValue, 0, 0, errorMsg)
			}
		}
		return nil
	}
}

func buildErrorMessage(messageAndArgs []any, err *V.ValidationError) string {
	var errorMsg string
	if len(messageAndArgs) == 1 {
		errorMsg = messageAndArgs[0].(string)
	} else if len(messageAndArgs) > 1 {
		errorMsg = fmt.Sprintf(messageAndArgs[0].(string), messageAndArgs[1:]...)
	}

	if len(errorMsg) > 0 {
		errorMsg = fmt.Sprintf("%s. %s", errorMsg, err)
	} else if len(errorMsg) == 0 {
		errorMsg = err.Error()
	}

	return errorMsg
}
