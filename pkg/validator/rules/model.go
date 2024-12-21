package rules

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
