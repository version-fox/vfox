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

	for _, forbidden := range []string{env.PidFlag, env.HookFlag, pathmeta.HookCurTmpPath, "_vfox_hook"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("IDE environment resolution script contains %q:\n%s", forbidden, got)
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
