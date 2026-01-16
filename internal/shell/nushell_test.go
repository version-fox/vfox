package shell

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

func TestExport(t *testing.T) {
	sep := string(os.PathListSeparator)

	var pathVarName string
	if runtime.GOOS == "windows" {
		pathVarName = "Path"
	} else {
		pathVarName = "PATH"
	}

	tests := []struct {
		name string
		envs env.Vars
		want nushellExportData
	}{
		{
			"Empty",
			env.Vars{},
			nushellExportData{
				EnvsToSet:   make(map[string]any),
				EnvsToUnset: make([]string, 0)},
		},
		{
			"SingleEnv",
			env.Vars{"FOO": newString("bar")},
			nushellExportData{
				EnvsToSet:   map[string]any{"FOO": "bar"},
				EnvsToUnset: make([]string, 0),
			},
		},
		{
			"MultipleEnvs",
			env.Vars{"FOO": newString("bar"), "BAZ": newString("qux")},
			nushellExportData{
				EnvsToSet:   map[string]any{"FOO": "bar", "BAZ": "qux"},
				EnvsToUnset: make([]string, 0),
			},
		},
		{
			"UnsetEnv",
			env.Vars{"FOO": nil},
			nushellExportData{
				EnvsToSet:   make(map[string]any),
				EnvsToUnset: []string{"FOO"},
			},
		},
		{
			"MixedEnvs",
			env.Vars{"FOO": newString("bar"), "BAZ": nil},
			nushellExportData{
				EnvsToSet:   map[string]any{"FOO": "bar"},
				EnvsToUnset: []string{"BAZ"},
			},
		},
		{
			"MultipleUnsetEnvs",
			env.Vars{"FOO": nil, "BAZ": nil},
			nushellExportData{
				EnvsToSet:   make(map[string]any),
				EnvsToUnset: []string{"FOO", "BAZ"},
			},
		},
		{
			"PathEnv",
			env.Vars{"PATH": newString("/path1" + sep + "/path2")},
			nushellExportData{
				EnvsToSet:   map[string]any{pathVarName: []any{"/path1", "/path2"}},
				EnvsToUnset: make([]string, 0),
			},
		},
		{
			"PathAndOtherEnv",
			env.Vars{
				"PATH": newString("/path1" + sep + "/path2" + sep + "/path3"),
				"FOO":  newString("bar"),
				"BAZ":  nil,
			},
			nushellExportData{
				EnvsToSet:   map[string]any{pathVarName: []any{"/path1", "/path2", "/path3"}, "FOO": "bar"},
				EnvsToUnset: []string{"BAZ"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runExportTest(t, test.envs, test.want)
		})
	}
}

func runExportTest(t *testing.T, envs env.Vars, want nushellExportData) {
	n := nushell{}
	got := n.Export(envs)
	var gotData nushellExportData
	err := json.Unmarshal([]byte(got), &gotData)
	if err != nil {
		t.Errorf("%s: error unmarshaling export data - %v", t.Name(), err)
		return
	}

	slices.Sort(want.EnvsToUnset)
	slices.Sort(gotData.EnvsToUnset)
	if !reflect.DeepEqual(gotData, want) {
		t.Errorf("%s: export data mismatch - want %v, got %v", t.Name(), want, gotData)
	}
}

func newString(s string) *string {
	return &s
}

func TestNushellActivate(t *testing.T) {
	tests := []struct {
		name           string
		config         ActivateConfig
		expectedInFile []string
		notInFile      []string
	}{
		{
			name: "EnablePidCheck true",
			config: ActivateConfig{
				SelfPath:       "/usr/local/bin/vfox",
				Args:           []string{t.TempDir()},
				EnablePidCheck: true,
			},
			expectedInFile: []string{
				"^'/usr/local/bin/vfox' activate nushell",
				"# Check if PID changed (e.g., in tmux new pane)",
				"if ($nu.pid != $env.__VFOX_PID) {",
				"$env.__VFOX_PID = $nu.pid",
				"^'/usr/local/bin/vfox' env -s nushell",
			},
			notInFile: []string{
				"{{if .EnablePidCheck}}",
				"{{end}}",
				"{{.SelfPath}}",
			},
		},
		{
			name: "EnablePidCheck false",
			config: ActivateConfig{
				SelfPath:       "/opt/vfox/bin/vfox",
				Args:           []string{t.TempDir()},
				EnablePidCheck: false,
			},
			expectedInFile: []string{
				"^'/opt/vfox/bin/vfox' activate nushell",
				"def --env updateVfoxEnvironment [] {",
				"^'/opt/vfox/bin/vfox' env -s nushell",
			},
			notInFile: []string{
				"{{if .EnablePidCheck}}",
				"{{end}}",
				"{{.SelfPath}}",
				"# Check if PID changed (e.g., in tmux new pane)",
				"if ($nu.pid != $env.__VFOX_PID) {",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := nushell{}
			script, err := n.Activate(tt.config)
			if err != nil {
				t.Fatalf("Activate() returned error: %v", err)
			}

			// Check that the returned script contains the source command
			if !strings.Contains(script, "source ($nu.default-config-dir | path join \"vfox.nu\")") {
				t.Errorf("Activate() script doesn't contain expected source command")
			}

			// Read the generated vfox.nu file
			targetPath := filepath.Join(tt.config.Args[0], "vfox.nu")
			content, err := os.ReadFile(targetPath)
			if err != nil {
				t.Fatalf("Failed to read generated file: %v", err)
			}

			fileContent := string(content)

			// Check expected content is present
			for _, expected := range tt.expectedInFile {
				if !strings.Contains(fileContent, expected) {
					t.Errorf("Generated file missing expected content: %q", expected)
				}
			}

			// Check that template directives are not present (should be evaluated)
			for _, notExpected := range tt.notInFile {
				if strings.Contains(fileContent, notExpected) {
					t.Errorf("Generated file contains unexpected content: %q", notExpected)
				}
			}
		})
	}
}
