//go:build !cgo

package shell

import (
	"os"
	"strings"
	"testing"
)

func TestGetShellName(t *testing.T) {
	// Save original SHELL env
	originalShell := os.Getenv("SHELL")
	defer func() {
		if originalShell != "" {
			os.Setenv("SHELL", originalShell)
		} else {
			os.Unsetenv("SHELL")
		}
	}()

	tests := []struct {
		name     string
		envShell string
		expected string
	}{
		{
			name:     "zsh from /bin/zsh",
			envShell: "/bin/zsh",
			expected: "zsh",
		},
		{
			name:     "bash from /usr/bin/bash",
			envShell: "/usr/bin/bash",
			expected: "bash",
		},
		{
			name:     "fish without path",
			envShell: "fish",
			expected: "fish",
		},
		{
			name:     "Windows cmd.exe",
			envShell: "/mnt/c/Windows/System32/cmd.exe",
			expected: "cmd",
		},
		{
			name:     "PowerShell",
			envShell: "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
			expected: "powershell",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SHELL", tt.envShell)
			result := GetShellName()
			if !strings.EqualFold(result, tt.expected) {
				t.Errorf("GetShellName() = %v, want %v", result, tt.expected)
			}
		})
	}

	// Test with no SHELL env (should fallback)
	t.Run("fallback", func(t *testing.T) {
		os.Unsetenv("SHELL")
		result := GetShellName()
		// Should return a non-empty string (fallback to "bash" or detected shell)
		if result == "" {
			t.Error("GetShellName() should not return empty string")
		}
	})
}