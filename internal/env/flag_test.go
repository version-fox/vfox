package env

import "testing"

func TestIsIDEEnvironmentResolution(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{
			name: "VS Code environment resolution",
			key:  "VSCODE_RESOLVING_ENVIRONMENT",
		},
		{
			name: "JetBrains environment reader",
			key:  "INTELLIJ_ENVIRONMENT_READER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, "1")
			if !IsIDEEnvironmentResolution() {
				t.Fatalf("IsIDEEnvironmentResolution() = false, want true when %s is set", tt.key)
			}
		})
	}

	t.Run("ordinary shell", func(t *testing.T) {
		t.Setenv("VSCODE_RESOLVING_ENVIRONMENT", "")
		t.Setenv("INTELLIJ_ENVIRONMENT_READER", "")
		if IsIDEEnvironmentResolution() {
			t.Fatal("IsIDEEnvironmentResolution() = true, want false without IDE env markers")
		}
	})
}

func TestIsInheritedHookSessionDetectsPidMismatchWithoutShellFlag(t *testing.T) {
	t.Setenv(PidFlag, "0")
	t.Setenv(HookFlag, "")

	if !IsInheritedHookSession() {
		t.Fatal("IsInheritedHookSession() = false, want true when inherited pid differs from parent process")
	}
}
