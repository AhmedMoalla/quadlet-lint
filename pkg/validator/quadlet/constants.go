package quadlet

var (
	// Supported keys in "Container" group
	supportedContainerKeys = map[string]bool{
		KeyAddCapability:         true,
		KeyAddDevice:             true,
		KeyAddHost:               true,
		KeyAnnotation:            true,
		KeyAutoUpdate:            true,
		KeyCgroupsMode:           true,
		KeyContainerName:         true,
		KeyContainersConfModule:  true,
		KeyDNS:                   true,
		KeyDNSOption:             true,
		KeyDNSSearch:             true,
		KeyDropCapability:        true,
		KeyEnvironment:           true,
		KeyEnvironmentFile:       true,
		KeyEnvironmentHost:       true,
		KeyEntrypoint:            true,
		KeyExec:                  true,
		KeyExposeHostPort:        true,
		KeyGIDMap:                true,
		KeyGlobalArgs:            true,
		KeyGroup:                 true,
		KeyGroupAdd:              true,
		KeyHealthCmd:             true,
		KeyHealthInterval:        true,
		KeyHealthOnFailure:       true,
		KeyHealthLogDestination:  true,
		KeyHealthMaxLogCount:     true,
		KeyHealthMaxLogSize:      true,
		KeyHealthRetries:         true,
		KeyHealthStartPeriod:     true,
		KeyHealthStartupCmd:      true,
		KeyHealthStartupInterval: true,
		KeyHealthStartupRetries:  true,
		KeyHealthStartupSuccess:  true,
		KeyHealthStartupTimeout:  true,
		KeyHealthTimeout:         true,
		KeyHostName:              true,
		KeyIP6:                   true,
		KeyIP:                    true,
		KeyImage:                 true,
		KeyLabel:                 true,
		KeyLogDriver:             true,
		KeyLogOpt:                true,
		KeyMask:                  true,
		KeyMount:                 true,
		KeyNetwork:               true,
		KeyNetworkAlias:          true,
		KeyNoNewPrivileges:       true,
		KeyNotify:                true,
		KeyPidsLimit:             true,
		KeyPod:                   true,
		KeyPodmanArgs:            true,
		KeyPublishPort:           true,
		KeyPull:                  true,
		KeyReadOnly:              true,
		KeyReadOnlyTmpfs:         true,
		KeyRemapGid:              true,
		KeyRemapUid:              true,
		KeyRemapUidSize:          true,
		KeyRemapUsers:            true,
		KeyRootfs:                true,
		KeyRunInit:               true,
		KeySeccompProfile:        true,
		KeySecret:                true,
		KeySecurityLabelDisable:  true,
		KeySecurityLabelFileType: true,
		KeySecurityLabelLevel:    true,
		KeySecurityLabelNested:   true,
		KeySecurityLabelType:     true,
		KeyServiceName:           true,
		KeyShmSize:               true,
		KeyStopSignal:            true,
		KeyStartWithPod:          true,
		KeyStopTimeout:           true,
		KeySubGIDMap:             true,
		KeySubUIDMap:             true,
		KeySysctl:                true,
		KeyTimezone:              true,
		KeyTmpfs:                 true,
		KeyUIDMap:                true,
		KeyUlimit:                true,
		KeyUnmask:                true,
		KeyUser:                  true,
		KeyUserNS:                true,
		KeyVolatileTmp:           true,
		KeyVolume:                true,
		KeyWorkingDir:            true,
	}

	// Supported keys in "Volume" group
	supportedVolumeKeys = map[string]bool{
		KeyContainersConfModule: true,
		KeyCopy:                 true,
		KeyDevice:               true,
		KeyDriver:               true,
		KeyGlobalArgs:           true,
		KeyGroup:                true,
		KeyImage:                true,
		KeyLabel:                true,
		KeyOptions:              true,
		KeyPodmanArgs:           true,
		KeyServiceName:          true,
		KeyType:                 true,
		KeyUser:                 true,
		KeyVolumeName:           true,
	}

	// Supported keys in "Network" group
	supportedNetworkKeys = map[string]bool{
		KeyLabel:                true,
		KeyDNS:                  true,
		KeyContainersConfModule: true,
		KeyGlobalArgs:           true,
		KeyDisableDNS:           true,
		KeyDriver:               true,
		KeyGateway:              true,
		KeyIPAMDriver:           true,
		KeyIPRange:              true,
		KeyIPv6:                 true,
		KeyInternal:             true,
		KeyNetworkName:          true,
		KeyOptions:              true,
		KeyServiceName:          true,
		KeySubnet:               true,
		KeyPodmanArgs:           true,
	}

	// Supported keys in "Kube" group
	supportedKubeKeys = map[string]bool{
		KeyAutoUpdate:           true,
		KeyConfigMap:            true,
		KeyContainersConfModule: true,
		KeyExitCodePropagation:  true,
		KeyGlobalArgs:           true,
		KeyKubeDownForce:        true,
		KeyLogDriver:            true,
		KeyLogOpt:               true,
		KeyNetwork:              true,
		KeyPodmanArgs:           true,
		KeyPublishPort:          true,
		KeyRemapGid:             true,
		KeyRemapUid:             true,
		KeyRemapUidSize:         true,
		KeyRemapUsers:           true,
		KeyServiceName:          true,
		KeySetWorkingDirectory:  true,
		KeyUserNS:               true,
		KeyYaml:                 true,
	}

	// Supported keys in "Image" group
	supportedImageKeys = map[string]bool{
		KeyAllTags:              true,
		KeyArch:                 true,
		KeyAuthFile:             true,
		KeyCertDir:              true,
		KeyContainersConfModule: true,
		KeyCreds:                true,
		KeyDecryptionKey:        true,
		KeyGlobalArgs:           true,
		KeyImage:                true,
		KeyImageTag:             true,
		KeyOS:                   true,
		KeyPodmanArgs:           true,
		KeyServiceName:          true,
		KeyTLSVerify:            true,
		KeyVariant:              true,
	}

	// Supported keys in "Build" group
	supportedBuildKeys = map[string]bool{
		KeyAnnotation:           true,
		KeyArch:                 true,
		KeyAuthFile:             true,
		KeyContainersConfModule: true,
		KeyDNS:                  true,
		KeyDNSOption:            true,
		KeyDNSSearch:            true,
		KeyEnvironment:          true,
		KeyFile:                 true,
		KeyForceRM:              true,
		KeyGlobalArgs:           true,
		KeyGroupAdd:             true,
		KeyImageTag:             true,
		KeyLabel:                true,
		KeyNetwork:              true,
		KeyPodmanArgs:           true,
		KeyPull:                 true,
		KeySecret:               true,
		KeyServiceName:          true,
		KeySetWorkingDirectory:  true,
		KeyTarget:               true,
		KeyTLSVerify:            true,
		KeyVariant:              true,
		KeyVolume:               true,
	}

	// Supported keys in "Pod" group
	supportedPodKeys = map[string]bool{
		KeyAddHost:              true,
		KeyContainersConfModule: true,
		KeyDNS:                  true,
		KeyDNSOption:            true,
		KeyDNSSearch:            true,
		KeyGIDMap:               true,
		KeyGlobalArgs:           true,
		KeyIP:                   true,
		KeyIP6:                  true,
		KeyNetwork:              true,
		KeyNetworkAlias:         true,
		KeyPodName:              true,
		KeyPodmanArgs:           true,
		KeyPublishPort:          true,
		KeyRemapGid:             true,
		KeyRemapUid:             true,
		KeyRemapUidSize:         true,
		KeyRemapUsers:           true,
		KeyServiceName:          true,
		KeySubGIDMap:            true,
		KeySubUIDMap:            true,
		KeyUIDMap:               true,
		KeyUserNS:               true,
		KeyVolume:               true,
	}

	// Supported keys in "Quadlet" group
	supportedQuadletKeys = map[string]bool{
		KeyDefaultDependencies: true,
	}
)

// All the supported quadlet keys
const (
	KeyAddCapability         = "AddCapability"
	KeyAddDevice             = "AddDevice"
	KeyAddHost               = "AddHost"
	KeyAllTags               = "AllTags"
	KeyAnnotation            = "Annotation"
	KeyArch                  = "Arch"
	KeyAuthFile              = "AuthFile"
	KeyAutoUpdate            = "AutoUpdate"
	KeyCertDir               = "CertDir"
	KeyCgroupsMode           = "CgroupsMode"
	KeyConfigMap             = "ConfigMap"
	KeyContainerName         = "ContainerName"
	KeyContainersConfModule  = "ContainersConfModule"
	KeyCopy                  = "Copy"
	KeyCreds                 = "Creds"
	KeyDecryptionKey         = "DecryptionKey"
	KeyDefaultDependencies   = "DefaultDependencies"
	KeyDevice                = "Device"
	KeyDisableDNS            = "DisableDNS"
	KeyDNS                   = "DNS"
	KeyDNSOption             = "DNSOption"
	KeyDNSSearch             = "DNSSearch"
	KeyDriver                = "Driver"
	KeyDropCapability        = "DropCapability"
	KeyEntrypoint            = "Entrypoint"
	KeyEnvironment           = "Environment"
	KeyEnvironmentFile       = "EnvironmentFile"
	KeyEnvironmentHost       = "EnvironmentHost"
	KeyExec                  = "Exec"
	KeyExitCodePropagation   = "ExitCodePropagation"
	KeyExposeHostPort        = "ExposeHostPort"
	KeyFile                  = "File"
	KeyForceRM               = "ForceRM"
	KeyGateway               = "Gateway"
	KeyGIDMap                = "GIDMap"
	KeyGlobalArgs            = "GlobalArgs"
	KeyGroup                 = "Group"
	KeyGroupAdd              = "GroupAdd"
	KeyHealthCmd             = "HealthCmd"
	KeyHealthInterval        = "HealthInterval"
	KeyHealthLogDestination  = "HealthLogDestination"
	KeyHealthMaxLogCount     = "HealthMaxLogCount"
	KeyHealthMaxLogSize      = "HealthMaxLogSize"
	KeyHealthOnFailure       = "HealthOnFailure"
	KeyHealthRetries         = "HealthRetries"
	KeyHealthStartPeriod     = "HealthStartPeriod"
	KeyHealthStartupCmd      = "HealthStartupCmd"
	KeyHealthStartupInterval = "HealthStartupInterval"
	KeyHealthStartupRetries  = "HealthStartupRetries"
	KeyHealthStartupSuccess  = "HealthStartupSuccess"
	KeyHealthStartupTimeout  = "HealthStartupTimeout"
	KeyHealthTimeout         = "HealthTimeout"
	KeyHostName              = "HostName"
	KeyImage                 = "Image"
	KeyImageTag              = "ImageTag"
	KeyInternal              = "Internal"
	KeyIP                    = "IP"
	KeyIP6                   = "IP6"
	KeyIPAMDriver            = "IPAMDriver"
	KeyIPRange               = "IPRange"
	KeyIPv6                  = "IPv6"
	KeyKubeDownForce         = "KubeDownForce"
	KeyLabel                 = "Label"
	KeyLogDriver             = "LogDriver"
	KeyLogOpt                = "LogOpt"
	KeyMask                  = "Mask"
	KeyMount                 = "Mount"
	KeyNetwork               = "Network"
	KeyNetworkAlias          = "NetworkAlias"
	KeyNetworkName           = "NetworkName"
	KeyNoNewPrivileges       = "NoNewPrivileges"
	KeyNotify                = "Notify"
	KeyOptions               = "Options"
	KeyOS                    = "OS"
	KeyPidsLimit             = "PidsLimit"
	KeyPod                   = "Pod"
	KeyPodmanArgs            = "PodmanArgs"
	KeyPodName               = "PodName"
	KeyPublishPort           = "PublishPort"
	KeyPull                  = "Pull"
	KeyReadOnly              = "ReadOnly"
	KeyReadOnlyTmpfs         = "ReadOnlyTmpfs"
	KeyRemapGid              = "RemapGid"     // deprecated
	KeyRemapUid              = "RemapUid"     // deprecated
	KeyRemapUidSize          = "RemapUidSize" // deprecated
	KeyRemapUsers            = "RemapUsers"   // deprecated
	KeyRootfs                = "Rootfs"
	KeyRunInit               = "RunInit"
	KeySeccompProfile        = "SeccompProfile"
	KeySecret                = "Secret"
	KeySecurityLabelDisable  = "SecurityLabelDisable"
	KeySecurityLabelFileType = "SecurityLabelFileType"
	KeySecurityLabelLevel    = "SecurityLabelLevel"
	KeySecurityLabelNested   = "SecurityLabelNested"
	KeySecurityLabelType     = "SecurityLabelType"
	KeyServiceName           = "ServiceName"
	KeySetWorkingDirectory   = "SetWorkingDirectory"
	KeyShmSize               = "ShmSize"
	KeyStartWithPod          = "StartWithPod"
	KeyStopSignal            = "StopSignal"
	KeyStopTimeout           = "StopTimeout"
	KeySubGIDMap             = "SubGIDMap"
	KeySubnet                = "Subnet"
	KeySubUIDMap             = "SubUIDMap"
	KeySysctl                = "Sysctl"
	KeyTarget                = "Target"
	KeyTimezone              = "Timezone"
	KeyTLSVerify             = "TLSVerify"
	KeyTmpfs                 = "Tmpfs"
	KeyType                  = "Type"
	KeyUIDMap                = "UIDMap"
	KeyUlimit                = "Ulimit"
	KeyUnmask                = "Unmask"
	KeyUser                  = "User"
	KeyUserNS                = "UserNS"
	KeyVariant               = "Variant"
	KeyVolatileTmp           = "VolatileTmp" // deprecated
	KeyVolume                = "Volume"
	KeyVolumeName            = "VolumeName"
	KeyWorkingDir            = "WorkingDir"
	KeyYaml                  = "Yaml"
	// service group keys

	KeyKillMode = "KillMode"
)

// Names of commonly used systemd/quadlet group names
const (
	ContainerGroup = "Container"
	InstallGroup   = "Install"
	KubeGroup      = "Kube"
	NetworkGroup   = "Network"
	PodGroup       = "Pod"
	ServiceGroup   = "Service"
	UnitGroup      = "Unit"
	VolumeGroup    = "Volume"
	ImageGroup     = "Image"
	BuildGroup     = "Build"
	QuadletGroup   = "Quadlet"
)
