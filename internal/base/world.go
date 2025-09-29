package base

type HookFunc struct {
	Name     string
	Required bool
	Filename string
}

var (
	// HookFuncMap is a map of built-in hook functions.
	HookFuncMap = map[string]HookFunc{
		"Available":       {Name: "Available", Required: true, Filename: "available"},
		"PreInstall":      {Name: "PreInstall", Required: true, Filename: "pre_install"},
		"EnvKeys":         {Name: "EnvKeys", Required: true, Filename: "env_keys"},
		"PostInstall":     {Name: "PostInstall", Required: false, Filename: "post_install"},
		"PreUse":          {Name: "PreUse", Required: false, Filename: "pre_use"},
		"ParseLegacyFile": {Name: "ParseLegacyFile", Required: false, Filename: "parse_legacy_file"},
		"PreUninstall":    {Name: "PreUninstall", Required: false, Filename: "pre_uninstall"},
	}
)

const (
	PluginObjKey    = "PLUGIN"
	NavigatorObjKey = "VFOX_NAVIGATOR"
	OsType          = "OS_TYPE"
	ArchType        = "ARCH_TYPE"
	Runtime         = "RUNTIME"
)
