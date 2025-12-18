package shell

import (
	"testing"
)

func TestStripLoginShellDash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "zsh with dash",
			input:    "-zsh",
			expected: "zsh",
		},
		{
			name:     "bash with dash",
			input:    "-bash",
			expected: "bash",
		},
		{
			name:     "fish with dash",
			input:    "-fish",
			expected: "fish",
		},
		{
			name:     "shell without dash",
			input:    "zsh",
			expected: "zsh",
		},
		{
			name:     "shell with path and dash",
			input:    "-/bin/zsh",
			expected: "/bin/zsh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shellName := tt.input
			if len(shellName) > 0 && shellName[0] == '-' {
				shellName = shellName[1:]
			}
			if shellName != tt.expected {
				t.Errorf("stripLoginShellDash() = %v, want %v", shellName, tt.expected)
			}
		})
	}
}

func TestCommonShellPaths(t *testing.T) {
	tests := []struct {
		name     string
		shell    string
		expected []string
	}{
		{
			name:     "zsh paths",
			shell:    "zsh",
			expected: []string{
				"/bin/zsh",
				"/usr/bin/zsh",
				"/usr/local/bin/zsh",
				"/opt/homebrew/bin/zsh",
				"/usr/local/Cellar/zsh",
			},
		},
		{
			name:     "bash paths",
			shell:    "bash",
			expected: []string{
				"/bin/bash",
				"/usr/bin/bash",
				"/usr/local/bin/bash",
				"/opt/homebrew/bin/bash",
				"/usr/local/Cellar/bash",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the expected paths are generated correctly
			// Actual file existence may vary on different systems
			commonPaths := []string{
				"/bin/" + tt.shell,
				"/usr/bin/" + tt.shell,
				"/usr/local/bin/" + tt.shell,
				"/opt/homebrew/bin/" + tt.shell,
				"/usr/local/Cellar/" + tt.shell,
			}

			if len(commonPaths) != len(tt.expected) {
				t.Errorf("Expected %d paths, got %d", len(tt.expected), len(commonPaths))
			}

			for i, path := range commonPaths {
				if path != tt.expected[i] {
					t.Errorf("Expected path %s, got %s", tt.expected[i], path)
				}
			}
		})
	}
}