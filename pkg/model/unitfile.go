package model

var (
	UnitTypeContainer = UnitType{Name: "container", Ext: ".container"}
	UnitTypeVolume    = UnitType{Name: "volume", Ext: ".volume"}
	UnitTypeKube      = UnitType{Name: "kube", Ext: ".kube"}
	UnitTypeNetwork   = UnitType{Name: "network", Ext: ".network"}
	UnitTypeImage     = UnitType{Name: "image", Ext: ".image"}
	UnitTypeBuild     = UnitType{Name: "build", Ext: ".build"}
	UnitTypePod       = UnitType{Name: "pod", Ext: ".pod"}
)

type UnitFile interface {
	FileName() string
	UnitType() UnitType
	Lookup(field Field) (LookupResult, bool)
	HasGroup(groupName string) bool
	ListGroups() []string
	ListKeys(groupName string) []UnitKey
	HasKey(field Field) bool
	HasValue(field Field) bool
}

type UnitType struct {
	Name string
	Ext  string
}

type UnitKey struct {
	Key  string
	Line int
}

type UnitValue struct {
	Key    string
	Value  string
	Line   int
	Column int
}

func (v UnitValue) String() string {
	return v.Value
}

type LookupResult interface {
	Values() []UnitValue

	IntValue() int
	BoolValue() bool
	Value() (UnitValue, bool)
}
