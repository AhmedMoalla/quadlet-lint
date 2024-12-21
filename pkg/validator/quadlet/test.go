package quadlet

import (
	"fmt"
	"github.com/AhmedMoalla/quadlet-lint/pkg/parser"
	V "github.com/AhmedMoalla/quadlet-lint/pkg/validator"
)

type Groups struct {
	Container
}

type Container struct {
	Rootfs []Rule
	Image  []Rule
}

type Field struct {
	Group string
	Name  string
}

func (f Field) String() string {
	return fmt.Sprintf("%s.%s", f.Group, f.Name)
}

var (
	Image  = Field{Group: "Container", Name: "Image"}
	Rootfs = Field{Group: "Container", Name: "Rootfs"}
)

type Rule = func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError

func RequiredIfNotPresent(other Field) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
		if !unit.HasValue(other.Group, other.Name) && !unit.HasValue(field.Group, field.Name) {
			return V.ErrSlice(validator.Name(), V.RequiredKey, 0, 0,
				fmt.Sprintf("at least one of these keys is required: %s, %s", field, other))
		}

		return nil
	}
}

func ConflictsWith(other Field) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {
		if unit.HasValue(other.Group, other.Name) && unit.HasValue(field.Group, field.Name) {
			return V.ErrSlice(validator.Name(), V.KeyConflict, 0, 0,
				fmt.Sprintf("the keys %s, %s cannot be specified together", field, other))
		}
		return nil
	}
}

func CanReference(unitType parser.UnitType) Rule {
	return func(validator V.Validator, unit parser.UnitFile, field Field) []V.ValidationError {

		return nil
	}
}

func test() {
	rules := Groups{
		Container: Container{
			Rootfs: []Rule{
				RequiredIfNotPresent(Image),
				ConflictsWith(Image),
				CanReference(parser.UnitTypeImage),
			},
			Image: []Rule{
				RequiredIfNotPresent(Rootfs),
			},
		},
	}

	fmt.Println(rules)
	//rules.Container = {
	//	Rootfs = [
	//		RequiredIfNotPresent(Image),
	//		ConflictsWith(Image),
	//		CanReference(UnitTypeImage, UnitTypeBuild),
	//	]
	//	Image = [
	//		ImageNotAmbiguous,
	//		RequiredIfNotPresent(Rootfs),
	//		ConflictsWith(Rootfs),
	//		CanReference(UnitTypeImage, UnitTypeBuild),
	//	]
	//	Network = [
	//		MultipleOccurences, => Should not be a rule as there is nothing to validate (Define it on the Field struct)
	//		CanReference(UnitTypeNetwork, UnitTypeContainer),
	//		HasFormat(NetworkFormat),
	//	]
	//	Group = [
	//		DependsOn(User),
	//	]
	//	RemapUid = [
	//		Deprecated,
	//	]
	//	ExposeHostPort = [
	//		MatchesRegex(`\d+(-\d+)?(/udp|/tcp)?$`), // or HasFormat(ExposeHostPortFormat) ?
	//	]
	//	Pod = [
	//		HasSuffix(".pod"),
	//		CanReference(UnitTypePod)
	//	]
	//}
	//
	//Group.Service = {
	//	KillMode = [
	//		AllowedValues("mixed", "control-group")
	//	]
	//	Type = [
	//		AllowedValues("notify", "oneshot")
	//	]
	//}
}
