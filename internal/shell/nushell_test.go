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
			"PathNil",
			env.Vars{"PATH": nil},
			nushellExportData{
				EnvsToSet:   map[string]any{pathVarName: []any{}},
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

			// Verify that get -o is used (not get -i)
			if strings.Contains(contentStr, "get -i hooks.pre_prompt") {
				t.Error("Found outdated 'get -i', should be 'get -o'")
			}
			if !strings.Contains(contentStr, "get -o hooks.pre_prompt") {
				t.Error("Missing correct 'get -o hooks.pre_prompt'")
			}

			// Verify guarded environment variable handling
			if !strings.Contains(contentStr, "if not ($envData.envsToSet | is-empty)") {
				t.Error("Missing envsToSet emptiness check")
			}
			if !strings.Contains(contentStr, "if not ($envData.envsToUnset | is-empty)") {
				t.Error("Missing envsToUnset emptiness check")
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

func TestNushellActivateDirect(t *testing.T) {
	n := nushell{}

	tests := []struct {
		name           string
		enablePidCheck bool
		expectPidBlock bool
	}{
		{
			name:           "Direct activate with PID check",
			enablePidCheck: true,
			expectPidBlock: true,
		},
		{
			name:           "Direct activate without PID check",
			enablePidCheck: false,
			expectPidBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := n.Activate(ActivateConfig{
				SelfPath:       "/usr/local/bin/vfox",
				Args:           nil,
				EnablePidCheck: tt.enablePidCheck,
			})
			if err != nil {
				t.Fatalf("Activate() failed: %v", err)
			}

			if script == "" {
				t.Fatal("Activate() returned empty script")
			}

			// Direct activation should return the script directly without sourcing
			if strings.Contains(script, "source ($nu.default-config-dir") {
				t.Error("Direct script should not contain source line")
			}
			if strings.Contains(script, "^'/usr/local/bin/vfox' activate nushell $nu.default-config-dir") {
				t.Error("Direct script should not contain self-re-activate line")
			}

			// Verify template evaluation
			if strings.Contains(script, "{{if .EnablePidCheck}}") || strings.Contains(script, "{{end}}") {
				t.Error("Template conditional directives were not evaluated")
			}
			if strings.Contains(script, "{{.SelfPath}}") {
				t.Error("Template variable '{{.SelfPath}}' was not evaluated")
			}

			// Verify self path and required nushell hooks
			if !strings.Contains(script, "/usr/local/bin/vfox") {
				t.Error("SelfPath missing from script")
			}
			if !strings.Contains(script, "export-env") {
				t.Error("Missing export-env block")
			}
			if !strings.Contains(script, "def --env updateVfoxEnvironment") {
				t.Error("Missing updateVfoxEnvironment function definition")
			}
			if !strings.Contains(script, "$env.config = ($env.config | upsert hooks.pre_prompt") {
				t.Error("Missing hooks.pre_prompt configuration")
			}
			if !strings.Contains(script, "get -o hooks.pre_prompt") {
				t.Error("Missing optional get flag for hooks.pre_prompt")
			}
			if !strings.Contains(script, "$env.__VFOX_SHELL = 'nushell'") {
				t.Error("Missing __VFOX_SHELL declaration")
			}
			if !strings.Contains(script, "if not ($envData.envsToSet | is-empty)") {
				t.Error("Missing envsToSet emptiness check")
			}
			if !strings.Contains(script, "if not ($envData.envsToUnset | is-empty)") {
				t.Error("Missing envsToUnset emptiness check")
			}

			hasPidCheck := strings.Contains(script, "# Check if PID changed") &&
				strings.Contains(script, "if ($nu.pid != $env.__VFOX_PID)")

			if tt.expectPidBlock && !hasPidCheck {
				t.Error("Expected PID check block to be present when EnablePidCheck is true")
			}
			if !tt.expectPidBlock && hasPidCheck {
				t.Error("Expected PID check block to be absent when EnablePidCheck is false")
			}
		})
	}
}

func TestNushellActivateErrors(t *testing.T) {
	n := nushell{}

	t.Run("MkdirAll error", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "regular_file")
		if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		// Passing filePath/child causes os.MkdirAll to fail because filePath is a regular file
		invalidDirPath := filepath.Join(filePath, "child")
		_, err := n.Activate(ActivateConfig{
			SelfPath: "/usr/bin/vfox",
			Args:     []string{invalidDirPath},
		})
		if err == nil {
			t.Fatal("Expected error from MkdirAll, got nil")
		}
		if !strings.Contains(err.Error(), "failed to create config directory") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("WriteFile error", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create a directory where the file would be written
		dirInPlaceOfFile := filepath.Join(tempDir, "vfox.nu")
		if err := os.Mkdir(dirInPlaceOfFile, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		_, err := n.Activate(ActivateConfig{
			SelfPath: "/usr/bin/vfox",
			Args:     []string{tempDir},
		})
		if err == nil {
			t.Fatal("Expected error from WriteFile, got nil")
		}
		if !strings.Contains(err.Error(), "failed to write file") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

func TestNushellActivateCreatesConfigDir(t *testing.T) {
	n := nushell{}
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "nested", "config")

	script, err := n.Activate(ActivateConfig{
		SelfPath: "/usr/bin/vfox",
		Args:     []string{configDir},
	})
	if err != nil {
		t.Fatalf("Activate() failed: %v", err)
	}
	if script == "" {
		t.Fatal("Activate() returned empty script")
	}

	targetPath := filepath.Join(configDir, "vfox.nu")
	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("Expected %s to exist, stat failed: %v", targetPath, err)
	}
}

// Nushell cannot evaluate what `vfox activate` prints: it is only a `source`
// line, and the exports in it never reach the session. So vfox.nu has to record
// the PATH baseline itself, and before the first `vfox env`, which compares the
// PATH it finds with it.
func TestNushellActivateRecordsTheBaseline(t *testing.T) {
	configPath := t.TempDir()
	if _, err := (nushell{}).Activate(ActivateConfig{SelfPath: "/usr/bin/vfox", Args: []string{configPath}}); err != nil {
		t.Fatalf("Activate() failed: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(configPath, "vfox.nu"))
	if err != nil {
		t.Fatalf("Failed to read vfox.nu: %v", err)
	}
	script := string(content)

	if strings.Contains(script, "{{") {
		t.Errorf("a template directive was not evaluated:\n%s", script)
	}

	assignment := "$env." + env.OrigPathFlag + " = "
	start := strings.Index(script, assignment)
	if start < 0 {
		t.Fatalf("vfox.nu does not set %s:\n%s", env.OrigPathFlag, script)
	}
	line, _, _ := strings.Cut(script[start:], "\n")
	line = strings.TrimRight(line, "\r")

	// It records the PATH of the session, the way the platform spells the variable, as one string.
	if want := "($env." + env.PathVarName + " | str join '" + string(os.PathListSeparator) + "')"; !strings.HasSuffix(line, want) {
		t.Errorf("the baseline is set by %q, want it to end with %q", line, want)
	}

	// The initial update, the last use of the function, calls `vfox env`.
	if initial := strings.LastIndex(script, "updateVfoxEnvironment"); start > initial {
		t.Errorf("the baseline is set after the first `vfox env` call:\n%s", script)
	}
	// And it is set inside export-env, which is what changes the session.
	if exportEnv := strings.Index(script, "export-env {"); exportEnv < 0 || start < exportEnv {
		t.Errorf("the baseline is set outside export-env, which would not reach the session:\n%s", script)
	}
}
