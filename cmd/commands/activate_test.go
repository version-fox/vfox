package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
)

func TestRenderActivateScriptForIDEEnvironmentResolutionSkipsHookSessionState(t *testing.T) {
	t.Setenv("VSCODE_RESOLVING_ENVIRONMENT", "1")

	path := "/tmp/bin"
	exportEnvs := env.Vars{
		"PATH": &path,
	}

	got, err := renderActivateScript("zsh", "/usr/bin/vfox", nil, exportEnvs, false)
	if err != nil {
		t.Fatalf("renderActivateScript() failed: %v", err)
	}

	if strings.Contains(got, "_vfox_hook") {
		t.Fatalf("IDE environment resolution script should not register hook:\n%s", got)
	}
	// Hook session vars must be unset (not just omitted) so they cannot leak
	// from the IDE's parent shell into every integrated terminal.
	for _, key := range []string{env.PidFlag, env.HookFlag, env.InitializedFlag, pathmeta.HookCurTmpPath} {
		want := "unset " + key + ";"
		if !strings.Contains(got, want) {
			t.Fatalf("IDE environment resolution script should unset %q, got:\n%s", key, got)
		}
		// And it must never re-export them.
		if strings.Contains(got, "export "+key+"=") {
			t.Fatalf("IDE environment resolution script should not export %q, got:\n%s", key, got)
		}
	}

	if !strings.Contains(got, `export PATH="/tmp/bin"`) {
		t.Fatalf("IDE environment resolution script should still export PATH, got:\n%s", got)
	}
}

func TestRenderActivateScriptPassesNushellConfigPath(t *testing.T) {
	configDir := t.TempDir()

	got, err := renderActivateScript("nushell", "/usr/bin/vfox", []string{configDir}, env.Vars{}, false)
	if err != nil {
		t.Fatalf("renderActivateScript() failed: %v", err)
	}

	if !strings.Contains(got, `source ($nu.default-config-dir | path join "vfox.nu")`) {
		t.Fatalf("renderActivateScript() returned unexpected Nushell source script:\n%s", got)
	}

	vfoxNu := filepath.Join(configDir, "vfox.nu")
	content, err := os.ReadFile(vfoxNu)
	if err != nil {
		t.Fatalf("expected generated %s: %v", vfoxNu, err)
	}
	if !strings.Contains(string(content), "^'/usr/bin/vfox' activate nushell $nu.default-config-dir") {
		t.Fatalf("generated vfox.nu does not reference self path:\n%s", string(content))
	}
}
