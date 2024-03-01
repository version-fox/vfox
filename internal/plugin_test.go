package internal

import (
	"testing"

	_ "embed"
)

//go:embed testdata/plugins/java.lua
var pluginContent string
var pluginPath = "testdata/plugins/java.lua"

func TestPlugin(t *testing.T) {
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
