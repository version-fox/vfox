package internal

import (
	"strings"
	"testing"

	_ "embed"

	"github.com/version-fox/vfox/internal/logger"
)

//go:embed testdata/plugins/java.lua
var pluginContent string
var pluginPath = "testdata/plugins/java.lua"

func setupSuite(tb testing.TB) func(tb testing.TB) {
	logger.SetLevel(logger.DebugLevel)

	return func(tb testing.TB) {
		logger.SetLevel(logger.InfoLevel)
	}
}

func TestPlugin(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	t.Run("Available", func(t *testing.T) {
		manager := NewSdkManager()
		plugin, err := NewLuaPlugin(pluginContent, pluginPath, manager)
		if err != nil {
			t.Fatal(err)
		}

		pkgs, err := plugin.Available()
		if err != nil {
			t.Fatal(err)
		}

		if len(pkgs) != 1 {
			t.Errorf("expected 1 package, got %d", len(pkgs))
		}
	})

	t.Run("PreInstall", func(t *testing.T) {
		manager := NewSdkManager()
		plugin, err := NewLuaPlugin(pluginContent, pluginPath, manager)
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
		manager := NewSdkManager()

		plugin, err := NewLuaPlugin(pluginContent, pluginPath, manager)
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

		javaHome := keys["JAVA_HOME"]
		if *javaHome == "" {
			t.Errorf("expected JAVA_HOME to be set, got '%s'", *javaHome)
		}
		path := keys["PATH"]
		if *path == "" {
			t.Errorf("expected PATH to be set, got '%s'", *path)
		}

		if !strings.HasSuffix(*path, "/bin") {
			t.Errorf("expected PATH to end with '/bin', got '%s'", *path)
		}
	})

	t.Run("PreUse", func(t *testing.T) {
		manager := NewSdkManager()

		plugin, err := NewLuaPlugin(pluginContent, pluginPath, manager)

		inputVersion := Version("20.0")
		previousVersion := Version("21.0")
		cwd := "/home/user"

		if err != nil {
			t.Fatal(err)
		}
		pkgs, err := plugin.Available()
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
}
