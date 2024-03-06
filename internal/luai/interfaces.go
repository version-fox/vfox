package luai

type LuaCheckSum struct {
	Sha256 string `luai:"sha256"`
	Sha512 string `luai:"sha512"`
	Sha1   string `luai:"sha1"`
	Md5    string `luai:"md5"`
}

type LuaSDKInfo struct {
	Name    string `luai:"name"`
	Version string `luai:"version"`
	Path    string `luai:"path"`
	Note    string `luai:"note"`
}

type AvailableHookCtx struct {
	RuntimeVersion string `luai:"runtimeVersion"`
}

type AvailableHookResultItem struct {
	Version string `luai:"version"`
	Note    string `luai:"note"`

	Addition []*LuaSDKInfo `luai:"addition"`
}

type PreInstallHookCtx struct {
	Version        string `luai:"version"`
	RuntimeVersion string `luai:"runtimeVersion"`
}

type PreInstallHookResultAdditionItem struct {
	Name string `luai:"name"`
	Url  string `luai:"url"`
	LuaCheckSum
}

type PreInstallHookResult struct {
	Version string `luai:"version"`
	Url     string `luai:"url"`
	LuaCheckSum

	Addition []*LuaSDKInfo `luai:"addition"`
}

type PreUseHookCtx struct {
	RuntimeVersion  string                 `luai:"runtimeVersion"`
	Cwd             string                 `luai:"cwd"`
	Scope           string                 `luai:"scope"`
	Version         string                 `luai:"version"`
	PreviousVersion string                 `luai:"previousVersion"`
	InstalledSdks   map[string]*LuaSDKInfo `luai:"installedSdks"`
}

type PreUseHookResult struct {
	Version string `luai:"version"`
}

type PostInstallHookCtx struct {
	RuntimeVersion string                 `luai:"runtimeVersion"`
	RootPath       string                 `luai:"rootPath"`
	SdkInfo        map[string]*LuaSDKInfo `luai:"sdkInfo"`
}

type EnvKeysHookCtx struct {
	RuntimeVersion string `luai:"runtimeVersion"`
	// TODO Will be deprecated in future versions
	Path    string                 `luai:"path"`
	SdkInfo map[string]*LuaSDKInfo `luai:"sdkInfo"`
}

type EnvKeysHookResultItem struct {
	Key   string `luai:"key"`
	Value string `luai:"value"`
}
