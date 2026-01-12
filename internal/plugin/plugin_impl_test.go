package plugin_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/plugin"
	"github.com/version-fox/vfox/internal/shared/logger"

	_ "embed"
)

var pluginPathWithMain = "testdata/plugins/java_with_main/"
var pluginPathWithMetadata = "testdata/plugins/java_with_metadata/"

func setupSuite(tb testing.TB) func(tb testing.TB) {
	logger.SetLevel(logger.DebugLevel)

	return func(tb testing.TB) {
		logger.SetLevel(logger.InfoLevel)
	}
}

func TestNewPluginWithMain(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	t.Run("NewLuaPlugin", func(t *testing.T) {
		manager, err := internal.NewSdkManager()
		if err != nil {
			t.Fatal(err)
		}
		defer manager.Close()

		plug, err := plugin.NewLuaPlugin(pluginPathWithMain, manager.RuntimeEnvContext)
		if err != nil {
			t.Fatal(err)
		}

		if plug == nil {
			t.Fatalf("expected plugin to be set, got nil")
		}

		if plug.InstalledPath != pluginPathWithMain {
			t.Errorf("expected filepath '%s', got '%s'", pluginPathWithMain, plug.InstalledPath)
		}

		if plug.Name != "java_with_main" {
			t.Errorf("expected name 'java', got '%s'", plug.Name)
		}

		if plug.Version != "0.0.1" {
			t.Errorf("expected version '0.0.1', got '%s'", plug.Version)
		}

		if plug.Description != "xxx" {
			t.Errorf("expected description 'xxx', got '%s'", plug.Description)
		}

		if plug.UpdateUrl != "{URL}/sdk.lua" {
			t.Errorf("expected update url '{URL}/sdk.lua', got '%s'", plug.UpdateUrl)
		}

		if plug.MinRuntimeVersion != "0.2.2" {
			t.Errorf("expected min runtime version '0.2.2', got '%s'", plug.MinRuntimeVersion)
		}
	})

	testHookFunc(t, func() (*internal.Manager, *plugin.Wrapper, error) {
		manager, err := internal.NewSdkManager()
		if err != nil {
			return nil, nil, err
		}
		plug, err := plugin.CreatePlugin(pluginPathWithMain, manager.RuntimeEnvContext)
		return manager, plug, err
	})

}

func TestNewLuaPluginWithMetadataAndHooks(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)
	t.Run("NewLuaPlugin", func(t *testing.T) {
		manager, err := internal.NewSdkManager()
		if err != nil {
			t.Fatal(err)
		}
		defer manager.Close()

		plug, err := plugin.CreatePlugin(pluginPathWithMetadata, manager.RuntimeEnvContext)
		if err != nil {
			t.Fatal(err)
		}

		if plug == nil {
			t.Fatalf("expected plugin to be set, got nil")
		}

		if plug.Name != "java_with_metadata" {
			t.Errorf("expected filename 'java', got '%s'", plug.Name)
		}

		if plug.InstalledPath != pluginPathWithMetadata {
			t.Errorf("expected filepath '%s', got '%s'", pluginPathWithMetadata, plug.InstalledPath)
		}

		if plug.Version != "0.0.1" {
			t.Errorf("expected version '0.0.1', got '%s'", plug.Version)
		}

		if plug.Description != "xxx" {
			t.Errorf("expected description 'xxx', got '%s'", plug.Description)
		}

		if plug.ManifestUrl != "manifest.json" {
			t.Errorf("expected manifest url 'manifest.json', got '%s'", plug.ManifestUrl)
		}

		if plug.MinRuntimeVersion != "0.2.2" {
			t.Errorf("expected min runtime version '0.2.2', got '%s'", plug.MinRuntimeVersion)
		}
		if !reflect.DeepEqual(plug.LegacyFilenames, []string{".node-version", ".nvmrc"}) {
			t.Errorf("expected legacy filenames '.node-version', '.nvmrc', got '%s'", plug.LegacyFilenames)
		}

		for _, hf := range plugin.HookFuncMap {
			if !plug.HasFunction(hf.Name) && hf.Required {
				t.Errorf("expected to have function %s", hf.Name)
			}
		}
	})
	testHookFunc(t, func() (*internal.Manager, *plugin.Wrapper, error) {
		manager, err := internal.NewSdkManager()
		if err != nil {
			return nil, nil, err
		}
		plug, err := plugin.CreatePlugin(pluginPathWithMetadata, manager.RuntimeEnvContext)
		return manager, plug, err
	})
}

func TestInvalidPluginName(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)
	manager, err := internal.NewSdkManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	_, err = plugin.CreatePlugin("testdata/plugins/invalid_name", manager.RuntimeEnvContext)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	t.Logf("error: %s", err.Error())
	if !strings.Contains(err.Error(), "invalid plugin name") {
		t.Errorf("expected error to contain 'invalid plugin name', got '%s'", err.Error())
	}
}

func TestMissingRequiredHook(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)
	manager, err := internal.NewSdkManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	_, err = plugin.CreatePlugin("testdata/plugins/missing_required_hook", manager.RuntimeEnvContext)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	t.Logf("error: %s", err.Error())
	if !strings.Contains(err.Error(), "[EnvKeys] function not found") {
		t.Errorf("expected error to contain '[EnvKeys] function not found', got '%s'", err.Error())
	}
}

func testHookFunc(t *testing.T, factory func() (*internal.Manager, *plugin.Wrapper, error)) {
	t.Helper()

	t.Run("PreInstall", func(t *testing.T) {
		_, plug, err := factory()
		if err != nil {
			t.Fatal(err)
		}

		pkg, err := plug.PreInstall(&plugin.PreInstallHookCtx{
			Version: "9.0.0",
		})
		if err != nil {
			t.Fatal(err)
		}

		Main := pkg.PreInstallPackageItem

		if Main.Version != "version" {
			t.Errorf("expected version 'version', got '%s'", Main.Version)
		}

		if Main.Path != "xxx" {
			t.Errorf("expected path 'xxx', got '%s'", Main.Path)
		}

		// checksum should be existed
		if Main.CheckSumItem == nil {
			t.Errorf("expected checksum to be set, got nil")
		}

		checksum := Main.Checksum()
		if checksum.Type != "sha256" {
			t.Errorf("expected checksum type 'sha256', got '%s'", checksum.Type)
		}

		if checksum.Value != "xxx" {
			t.Errorf("expected checksum value 'xxx', got '%s'", checksum.Value)
		}

		if len(pkg.Addition) != 1 {
			t.Errorf("expected 1 addition, got %d", len(pkg.Addition))
		}

		addition := pkg.Addition[0]

		if addition.Path != "xxx" {
			t.Errorf("expected path 'xxx', got '%s'", addition.Path)
		}

		if addition.CheckSumItem == nil {
			t.Errorf("expected checksum to be set, got nil")
		}
	})

	t.Run("EnvKeys", func(t *testing.T) {
		_, plug, err := factory()
		if err != nil {
			t.Fatal(err)
		}

		mainPath := "/path/to/java"
		items, err := plug.EnvKeys(&plugin.EnvKeysHookCtx{
			Path: mainPath, // This is ctx.path in Lua
			Main: &plugin.InstalledPackageItem{
				Name:    "java",
				Version: "1.0.0",
				Path:    mainPath,
			},
			SdkInfo: map[string]*plugin.InstalledPackageItem{
				"sdk-name": {
					Name:    "sdk-name",
					Version: "9.0.0",
					Path:    "/path/to/sdk",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Returned %d env items", len(items))
		for i, item := range items {
			t.Logf("Item %d: key=%s, value=%s", i, item.Key, item.Value)
		}

		javaHome := ""
		var paths []string
		for _, item := range items {
			if item.Key == "JAVA_HOME" {
				javaHome = item.Value
			}
			if item.Key == "PATH" {
				paths = append(paths, item.Value)
			}
		}

		if javaHome == "" {
			t.Errorf("expected JAVA_HOME to be set, got '%s'", javaHome)
		}

		if javaHome != mainPath {
			t.Errorf("expected JAVA_HOME to be '%s', got '%s'", mainPath, javaHome)
		}

		// Plugin returns 3 env items: JAVA_HOME, PATH (bin), PATH (bin2)
		if len(items) != 3 {
			t.Errorf("expected 3 env items, got %d", len(items))
		}

		// Check first PATH ends with /bin
		if len(paths) < 1 {
			t.Fatal("expected at least 1 PATH item")
		}
		firstPath := paths[0]
		expectedFirstPath := mainPath + "/bin"
		if firstPath != expectedFirstPath {
			t.Errorf("expected first PATH to be '%s', got '%s'", expectedFirstPath, firstPath)
		}
	})

	t.Run("PreUse", func(t *testing.T) {
		_, plug, err := factory()

		inputVersion := "20.0"
		previousVersion := "21.0"
		cwd := "/home/user"

		if err != nil {
			t.Fatal(err)
		}
		installedSdks := map[string]*plugin.InstalledPackageItem{
			"xxxx": {
				Name:    "xxxx",
				Version: "1.0.0",
				Path:    "/test/path",
			},
		}

		ctx := &plugin.PreUseHookCtx{
			Cwd:             cwd,
			Scope:           "global",
			Version:         inputVersion,
			PreviousVersion: previousVersion,
			InstalledSdks:   installedSdks,
		}
		result, err := plug.PreUse(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if result.Version != "9.9.9" {
			t.Errorf("expected version '9.9.9', got '%s'", result.Version)
		}

		ctx.Scope = "project"
		result, err = plug.PreUse(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if result.Version != "10.0.0" {
			t.Errorf("expected version '10.0.0', got '%s'", result.Version)
		}

		ctx.Scope = "session"
		result, err = plug.PreUse(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if result.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", result.Version)
		}
	})

	t.Run("ParseLegacyFile", func(t *testing.T) {
		_, plug, err := factory()
		if err != nil {
			t.Fatal(err)
		}
		if plug == nil {
			t.Fatal("factory returned nil plugin without error")
		}
		ctx := &plugin.ParseLegacyFileHookCtx{
			Filepath: "/path/to/legacy/.node-version",
			Filename: ".node-version",
			GetInstalledVersions: func() []string {
				return nil
			},
			Strategy: "specified",
		}
		result, err := plug.ParseLegacyFile(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if result.Version != "14.17.0" {
			t.Errorf("expected version '14.17.0', got '%s'", result.Version)
		}

		ctx.GetInstalledVersions = func() []string {
			return []string{"test"}
		}
		result, err = plug.ParseLegacyFile(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if result.Version != "check-installed" {
			t.Errorf("expected version 'check-installed', got '%s'", result.Version)
		}

		ctx.Filepath = "/path/to/legacy/.nvmrc"
		ctx.Filename = ".nvmrc"
		ctx.GetInstalledVersions = func() []string {
			return nil
		}
		result, err = plug.ParseLegacyFile(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if result.Version != "0.0.1" {
			t.Errorf("expected version '0.0.1', got '%s'", result.Version)
		}

		plug.LegacyFilenames = []string{}
		result, err = plug.ParseLegacyFile(ctx)
		if err != nil && result.Version != "" {
			t.Errorf("expected non version, got '%s'", result.Version)
		}

	})

	t.Run("PreUninstall", func(t *testing.T) {
		_, plug, err := factory()
		if err != nil {
			t.Fatal(err)
		}

		ctx := &plugin.PreUninstallHookCtx{
			Main: &plugin.InstalledPackageItem{
				Name:    "java",
				Version: "1.0.0",
				Path:    "/path/to/java",
			},
			SdkInfo: map[string]*plugin.InstalledPackageItem{
				"sdk-name": {
					Name:    "sdk-name",
					Version: "9.0.0",
					Path:    "/path/to/sdk",
				},
			},
		}

		err = plug.PreUninstall(ctx)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("LoadAllSdk", func(t *testing.T) {
		m, _, err := factory()
		if err != nil {
			t.Fatal(err)
		}

		firstSdkName := ""
		for i := 0; i < 10; i++ {
			sdks, err := m.LoadAllSdk()
			if err != nil {
				t.Fatal(err)
			}
			if len(sdks) != 0 && firstSdkName == "" {
				firstSdkName = sdks[0].Metadata().Name
			} else if len(sdks) != 0 {
				if sdks[0].Metadata().Name != firstSdkName {
					t.Errorf("expected sdk sort %v", sdks)
				}
			}
		}
	})
}
