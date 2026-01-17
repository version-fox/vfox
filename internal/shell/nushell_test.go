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

// TestNushellActivateWithPidCheck verifies that the nushell template correctly
// handles the EnablePidCheck conditional and fixes the get -o issue
func TestNushellActivateWithPidCheck(t *testing.T) {
	n := nushell{}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configPath, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	tests := []struct {
		name           string
		enablePidCheck bool
		expectPidBlock bool
	}{
		{
			name:           "EnablePidCheck true includes PID block",
			enablePidCheck: true,
			expectPidBlock: true,
		},
		{
			name:           "EnablePidCheck false excludes PID block",
			enablePidCheck: false,
			expectPidBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := n.Activate(ActivateConfig{
				SelfPath:       "/usr/bin/vfox",
				Args:           []string{configPath},
				EnablePidCheck: tt.enablePidCheck,
			})
			if err != nil {
				t.Fatalf("Activate() failed: %v", err)
			}

			// Check that the returned script is valid
			if script == "" {
				t.Fatal("Activate() returned empty script")
			}

			// Read the generated vfox.nu file
			vfoxNuPath := filepath.Join(configPath, "vfox.nu")
			content, err := os.ReadFile(vfoxNuPath)
			if err != nil {
				t.Fatalf("Failed to read vfox.nu: %v", err)
			}

			contentStr := string(content)

			// Verify that template directives are not present (they should be evaluated)
			if strings.Contains(contentStr, "{{if .EnablePidCheck}}") {
				t.Error("Template directive '{{if .EnablePidCheck}}' was not evaluated")
			}
			if strings.Contains(contentStr, "{{end}}") {
				t.Error("Template directive '{{end}}' was not evaluated")
			}
			if strings.Contains(contentStr, "{{.SelfPath}}") {
				t.Error("Template variable '{{.SelfPath}}' was not evaluated")
			}

			// Verify that SelfPath was replaced
			if !strings.Contains(contentStr, "/usr/bin/vfox") {
				t.Error("SelfPath was not properly replaced")
			}

			// Verify that get -i is used (not get -o)
			if strings.Contains(contentStr, "get -o hooks.pre_prompt") {
				t.Error("Found incorrect 'get -o', should be 'get -i'")
			}
			if !strings.Contains(contentStr, "get -i hooks.pre_prompt") {
				t.Error("Missing correct 'get -i hooks.pre_prompt'")
			}

			// Check for PID check block presence/absence
			hasPidCheck := strings.Contains(contentStr, "# Check if PID changed") &&
				strings.Contains(contentStr, "if ($nu.pid != $env.__VFOX_PID)")

			if tt.expectPidBlock && !hasPidCheck {
				t.Error("Expected PID check block to be present when EnablePidCheck is true")
			}
			if !tt.expectPidBlock && hasPidCheck {
				t.Error("Expected PID check block to be absent when EnablePidCheck is false")
			}
		})
	}
}
