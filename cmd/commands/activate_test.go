package commands

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
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

func TestActivateCmdIncludesShimDirs(t *testing.T) {
	f := newEnvCommandFixture(t)
	t.Chdir(f.work)
	// Pin the hook session to this process: activateCmd discards an
	// inherited session (IsInheritedHookSession), which would replace the
	// fixture session dir with a pid-based one when the outer shell
	// itself runs under vfox.
	t.Setenv(env.PidFlag, strconv.Itoa(os.Getppid()))
	// Keep IDE-resolution mode off so the full hook script is rendered.
	t.Setenv("VSCODE_RESOLVING_ENVIRONMENT", "")
	t.Setenv("INTELLIJ_ENVIRONMENT_READER", "")
	// activateCmd exports hook session vars; snapshot them to avoid leaking
	// into other tests in this process.
	snapshot := map[string]*string{}
	for _, key := range []string{env.HookFlag, env.PidFlag, env.InitializedFlag, pathmeta.HookCurTmpPath} {
		if v, ok := os.LookupEnv(key); ok {
			v := v
			snapshot[key] = &v
		} else {
			snapshot[key] = nil
		}
	}
	t.Cleanup(func() {
		for key, v := range snapshot {
			if v == nil {
				_ = os.Unsetenv(key)
			} else {
				_ = os.Setenv(key, *v)
			}
		}
	})

	oldWriter := Activate.Writer
	var buf bytes.Buffer
	Activate.Writer = &buf
	defer func() { Activate.Writer = oldWriter }()

	app := &cli.Command{Name: "vfox", Commands: []*cli.Command{Activate}}
	if err := app.Run(context.Background(), []string{"vfox", "activate", "bash"}); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	output := buf.String()
	// Stable shim dirs must be exported on PATH ahead of version bins.
	for _, shimDir := range []string{
		filepath.Join(f.work, ".vfox", "shims"),
		filepath.Join(filepath.Dir(f.state), "shims"),
		filepath.Join(f.home, "shims"),
	} {
		if !strings.Contains(output, shimDir) {
			t.Errorf("activate output missing shim dir %q:\n%s", shimDir, output)
		}
	}
}
