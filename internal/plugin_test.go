package internal

import (
	"github.com/version-fox/vfox/internal/util"
	"reflect"
	"strings"
	"testing"
	"time"

	_ "embed"

	"github.com/version-fox/vfox/internal/logger"
)

var pluginPathWithMain = "testdata/plugins/java_with_main/"
var pluginPathWithMetadata = "testdata/plugins/java_with_metadata/"

func setupSuite(tb testing.TB) func(tb testing.TB) {
	logger.SetLevel(logger.DebugLevel)

	return func(tb testing.TB) {
		logger.SetLevel(logger.InfoLevel)
	}
}

func TestNewLuaPluginWithMain(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	t.Run("NewLuaPlugin", func(t *testing.T) {
		manager := NewSdkManager()
		plugin, err := NewLuaPlugin(pluginPathWithMain, manager)
		if err != nil {
			t.Fatal(err)
		}

		if plugin == nil {
			t.Fatalf("expected plugin to be set, got nil")
		}

		if plugin.SdkName != "java_with_main" {
			t.Errorf("expected filename 'java', got '%s'", plugin.SdkName)
		}

		if plugin.Path != pluginPathWithMain {
			t.Errorf("expected filepath '%s', got '%s'", pluginPathWithMain, plugin.Path)
		}

		if plugin.Name != "java" {
			t.Errorf("expected name 'java', got '%s'", plugin.Name)
		}

		if plugin.Version != "0.0.1" {
			t.Errorf("expected version '0.0.1', got '%s'", plugin.Version)
		}

		if plugin.Description != "xxx" {
			t.Errorf("expected description 'xxx', got '%s'", plugin.Description)
		}

		if plugin.UpdateUrl != "{URL}/sdk.lua" {
			t.Errorf("expected update url '{URL}/sdk.lua', got '%s'", plugin.UpdateUrl)
		}

		if plugin.MinRuntimeVersion != "0.2.2" {
			t.Errorf("expected min runtime version '0.2.2', got '%s'", plugin.MinRuntimeVersion)
		}
	})

	testHookFunc(t, func() (*Manager, *LuaPlugin, error) {
		manager := NewSdkManager()
		plugin, err := NewLuaPlugin(pluginPathWithMain, manager)
		return manager, plugin, err
	})

}

func TestNewLuaPluginWithMetadataAndHooks(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)
	t.Run("NewLuaPlugin", func(t *testing.T) {
		manager := NewSdkManager()
		plugin, err := NewLuaPlugin(pluginPathWithMetadata, manager)
		if err != nil {
			t.Fatal(err)
		}

		if plugin == nil {
			t.Fatalf("expected plugin to be set, got nil")
		}

		if plugin.SdkName != "java_with_metadata" {
			t.Errorf("expected filename 'java', got '%s'", plugin.SdkName)
		}

		if plugin.Path != pluginPathWithMetadata {
			t.Errorf("expected filepath '%s', got '%s'", pluginPathWithMetadata, plugin.Path)
		}

		if plugin.Name != "java" {
			t.Errorf("expected name 'java', got '%s'", plugin.Name)
		}

		if plugin.Version != "0.0.1" {
			t.Errorf("expected version '0.0.1', got '%s'", plugin.Version)
		}

		if plugin.Description != "xxx" {
			t.Errorf("expected description 'xxx', got '%s'", plugin.Description)
		}

		if plugin.ManifestUrl != "manifest.json" {
			t.Errorf("expected manifest url 'manifest.json', got '%s'", plugin.ManifestUrl)
		}

		if plugin.MinRuntimeVersion != "0.2.2" {
			t.Errorf("expected min runtime version '0.2.2', got '%s'", plugin.MinRuntimeVersion)
		}
		if !reflect.DeepEqual(plugin.LegacyFilenames, []string{".node-version", ".nvmrc"}) {
			t.Errorf("expected legacy filenames '.node-version', '.nvmrc', got '%s'", plugin.LegacyFilenames)
		}

		for _, hf := range HookFuncMap {
			if !plugin.HasFunction(hf.Name) && hf.Required {
				t.Errorf("expected to have function %s", hf.Name)
			}
		}
	})
	testHookFunc(t, func() (*Manager, *LuaPlugin, error) {
		manager := NewSdkManager()
		plugin, err := NewLuaPlugin(pluginPathWithMetadata, manager)
		return manager, plugin, err
	})
}

func testHookFunc(t *testing.T, factory func() (*Manager, *LuaPlugin, error)) {
	t.Helper()

	t.Run("Available", func(t *testing.T) {
		m, plugin, err := factory()
		m.Config.Cache.AvailableHookDuration = time.Second * 10
		if err != nil {
			t.Fatal(err)
		}

		getResult := func() string {
			pkgs, err := plugin.Available([]string{})
			if err != nil {
				t.Fatal(err)
			}
			v := pkgs[0].Main.Note
			return v
		}
		version := getResult()
		for i := 0; i < 6; i++ {
			v := getResult()
			if version != v {
				t.Errorf("expected version '%s', got '%s'", version, v)
			}
			time.Sleep(time.Second)
		}

		time.Sleep(time.Second * 10)

		vv := getResult()
		if version == vv {
			t.Errorf("expected version to be different, got '%s'", vv)
		}
	})

	t.Run("PreInstall", func(t *testing.T) {
		_, plugin, err := factory()
		if err != nil {
			t.Fatal(err)
		}

		pkg, err := plugin.PreInstall(Version("9.0.0"))
		if err != nil {
			t.Fatal(err)
		}

		Main := pkg.Main

		if Main.Version != "version" {
			t.Errorf("expected version 'version', got '%s'", Main.Version)
		}

		if Main.Path != "xxx" {
			t.Errorf("expected path 'xxx', got '%s'", Main.Path)
		}

		// checksum should be existed
		if Main.Checksum == nil {
			t.Errorf("expected checksum to be set, got nil")
		}

		if Main.Checksum.Type != "sha256" {
			t.Errorf("expected checksum type 'sha256', got '%s'", Main.Checksum.Type)
		}

		if Main.Checksum.Value != "xxx" {
			t.Errorf("expected checksum value 'xxx', got '%s'", Main.Checksum.Value)
		}

		if len(pkg.Additions) != 1 {
			t.Errorf("expected 1 addition, got %d", len(pkg.Additions))
		}

		addition := pkg.Additions[0]

		if addition.Path != "xxx" {
			t.Errorf("expected path 'xxx', got '%s'", addition.Path)
		}

		if addition.Checksum == nil {
			t.Errorf("expected checksum to be set, got nil")
		}
	})

	t.Run("EnvKeys", func(t *testing.T) {
		_, plugin, err := factory()
		if err != nil {
			t.Fatal(err)
		}

		keys, err := plugin.EnvKeys(&Package{
			Main: &Info{
				Name:    "java",
				Version: "1.0.0",
				Path:    "/path/to/java",
				Note:    "xxxx",
			},
			Additions: []*Info{
				{
					Name:    "sdk-name",
					Version: "9.0.0",
					Path:    "/path/to/sdk",
					Note:    "xxxx",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		javaHome := keys.Variables["JAVA_HOME"]
		if *javaHome == "" {
			t.Errorf("expected JAVA_HOME to be set, got '%s'", *javaHome)
		}
		path := keys.Paths.Slice()
		if len(path) != 2 {
			t.Errorf("expected 2 paths, got %d", len(path))
		}

		if util.GetOSType() == "windows" {
			if !strings.HasSuffix(path[0], "\\bin") {
				t.Errorf("expected PATH to end with '\\bin', got '%s'", path[0])
			}
			if !strings.HasSuffix(path[1], "\\bin2") {
				t.Errorf("expected PATH to end with '\\bin2', got '%s'", path[1])
			}

		} else {
			if !strings.HasSuffix(path[0], "/bin") {
				t.Errorf("expected PATH to end with '/bin', got '%s'", path[0])
			}
			if !strings.HasSuffix(path[1], "/bin2") {
				t.Errorf("expected PATH to end with '/bin2', got '%s'", path[1])
			}
		}

	})

	t.Run("PreUse", func(t *testing.T) {
		_, plugin, err := factory()

		inputVersion := Version("20.0")
		previousVersion := Version("21.0")
		cwd := "/home/user"

		if err != nil {
			t.Fatal(err)
		}
		pkgs, err := plugin.Available([]string{})
		if err != nil {
			t.Fatal(err)
		}
		version, err := plugin.PreUse(inputVersion, previousVersion, Global, cwd, pkgs)
		if err != nil {
			t.Fatal(err)
		}

		if version != "9.9.9" {
			t.Errorf("expected version '9.9.9', got '%s'", version)
		}

		version, err = plugin.PreUse(inputVersion, previousVersion, Project, cwd, pkgs)
		if err != nil {
			t.Fatal(err)
		}

		if version != "10.0.0" {
			t.Errorf("expected version '10.0.0', got '%s'", version)
		}

		version, err = plugin.PreUse(inputVersion, previousVersion, Session, cwd, pkgs)
		if err != nil {
			t.Fatal(err)
		}

		if version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", version)
		}
	})

	t.Run("ParseLegacyFile", func(t *testing.T) {
		_, plugin, err := factory()
		version, err := plugin.ParseLegacyFile("/path/to/legacy/.node-version", func() []Version {
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if version != "14.17.0" {
			t.Errorf("expected version '14.17.0', got '%s'", version)
		}
		version, err = plugin.ParseLegacyFile("/path/to/legacy/.nvmrc", func() []Version {
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}

		if version != "0.0.1" {
			t.Errorf("expected version '0.0.1', got '%s'", version)
		}
		plugin.LegacyFilenames = []string{}
		version, err = plugin.ParseLegacyFile("/path/to/legacy/.nvmrc", func() []Version {
			return nil
		})
		if err != nil && version != "" {
			t.Errorf("expected non version, got '%s'", version)
		}

	})

	t.Run("PreUninstall", func(t *testing.T) {
		_, plugin, err := factory()
		if err != nil {
			t.Fatal(err)
		}

		err = plugin.PreUninstall(&Package{
			Main: &Info{
				Name:    "java",
				Version: "1.0.0",
				Path:    "/path/to/java",
				Note:    "xxxx",
			},
			Additions: []*Info{
				{
					Name:    "sdk-name",
					Version: "9.0.0",
					Path:    "/path/to/sdk",
					Note:    "xxxx",
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}
