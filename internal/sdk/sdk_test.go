package sdk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/base"
	"github.com/version-fox/vfox/internal/toolset"
)

func TestSdk_Unuse(t *testing.T) {
	// Setup temporary directory structure
	tempDir := t.TempDir()

	// Create PathMeta
	pathMeta := &PathMeta{
		User: UserPaths{
			Home: filepath.Join(tempDir, "user"),
		},
		Shared: SharedPaths{
			Installs: filepath.Join(tempDir, "shared", "installs"),
		},
		Working: WorkingPaths{
			Directory:   filepath.Join(tempDir, "project"),
			SessionShim: filepath.Join(tempDir, "session"),
			GlobalShim:  filepath.Join(tempDir, "user", "shims"),
		},
	}

	// Create directories
	os.MkdirAll(pathMeta.User.Home, 0755)
	os.MkdirAll(pathMeta.Shared.Installs, 0755)
	os.MkdirAll(pathMeta.Working.Directory, 0755)
	os.MkdirAll(pathMeta.Working.SessionShim, 0755)
	os.MkdirAll(pathMeta.Working.GlobalShim, 0755)

	// Create Manager
	manager := &Manager{
		PathMeta: pathMeta,
		openSdks: make(map[string]*impl),
	}

	// Create SDK
	sdk := &impl{
		Name:        "test-sdk",
		sdkManager:  manager,
		InstallPath: filepath.Join(pathMeta.Shared.Installs, "test-sdk"),
	}

	tests := []struct {
		name        string
		scope       base.UseScope
		setupFunc   func()             // Setup tool versions
		verifyFunc  func(t *testing.T) // Verify results
		expectError bool
	}{
		{
			name:  "Unuse global scope with SDK set",
			scope: base.Global,
			setupFunc: func() {
				// Setup global
				tv, _ := toolset.NewToolVersion(pathMeta.User.Home)
				tv.Record["test-sdk"] = "1.0.0"
				tv.Save()
				// Setup session (always processed)
				stv, _ := toolset.NewToolVersion(pathMeta.Working.SessionShim)
				stv.Record["test-sdk"] = "1.0.0"
				stv.Save()
			},
			verifyFunc: func(t *testing.T) {
				// Verify global removed
				tv, err := toolset.NewToolVersion(pathMeta.User.Home)
				if err != nil {
					t.Fatalf("Failed to read global tool version: %v", err)
				}
				if _, exists := tv.Record["test-sdk"]; exists {
					t.Errorf("Expected test-sdk to be removed from global tool version")
				}
				// Verify session removed
				stv, err := toolset.NewToolVersion(pathMeta.Working.SessionShim)
				if err != nil {
					t.Fatalf("Failed to read session tool version: %v", err)
				}
				if _, exists := stv.Record["test-sdk"]; exists {
					t.Errorf("Expected test-sdk to be removed from session tool version")
				}
			},
			expectError: false,
		},
		{
			name:  "Unuse project scope with SDK set",
			scope: base.Project,
			setupFunc: func() {
				// Setup project
				tv, _ := toolset.NewToolVersion(pathMeta.Working.Directory)
				tv.Record["test-sdk"] = "1.0.0"
				tv.Save()
				// Setup session
				stv, _ := toolset.NewToolVersion(pathMeta.Working.SessionShim)
				stv.Record["test-sdk"] = "1.0.0"
				stv.Save()
			},
			verifyFunc: func(t *testing.T) {
				// Verify project removed
				tv, err := toolset.NewToolVersion(pathMeta.Working.Directory)
				if err != nil {
					t.Fatalf("Failed to read project tool version: %v", err)
				}
				if _, exists := tv.Record["test-sdk"]; exists {
					t.Errorf("Expected test-sdk to be removed from project tool version")
				}
				// Verify session removed
				stv, err := toolset.NewToolVersion(pathMeta.Working.SessionShim)
				if err != nil {
					t.Fatalf("Failed to read session tool version: %v", err)
				}
				if _, exists := stv.Record["test-sdk"]; exists {
					t.Errorf("Expected test-sdk to be removed from session tool version")
				}
			},
			expectError: false,
		},
		{
			name:  "Unuse session scope",
			scope: base.Session,
			setupFunc: func() {
				// Setup session
				stv, _ := toolset.NewToolVersion(pathMeta.Working.SessionShim)
				stv.Record["test-sdk"] = "1.0.0"
				stv.Save()
			},
			verifyFunc: func(t *testing.T) {
				// Verify session removed
				stv, err := toolset.NewToolVersion(pathMeta.Working.SessionShim)
				if err != nil {
					t.Fatalf("Failed to read session tool version: %v", err)
				}
				if _, exists := stv.Record["test-sdk"]; exists {
					t.Errorf("Expected test-sdk to be removed from session tool version")
				}
			},
			expectError: false,
		},
		{
			name:  "Unuse when SDK not set in any scope",
			scope: base.Global,
			setupFunc: func() {
				// No setup
			},
			verifyFunc: func(t *testing.T) {
				// Verify no files created or modified
				if _, err := os.Stat(filepath.Join(pathMeta.User.Home, ".tool-versions")); !os.IsNotExist(err) {
					t.Errorf("Expected no global tool version file")
				}
				if _, err := os.Stat(filepath.Join(pathMeta.Working.SessionShim, ".tool-versions")); !os.IsNotExist(err) {
					t.Errorf("Expected no session tool version file")
				}
			},
			expectError: false,
		},
		{
			name:  "Unuse global when only session has SDK",
			scope: base.Global,
			setupFunc: func() {
				// Only session has SDK
				stv, _ := toolset.NewToolVersion(pathMeta.Working.SessionShim)
				stv.Record["test-sdk"] = "1.0.0"
				stv.Save()
			},
			verifyFunc: func(t *testing.T) {
				// Verify global unchanged (no file)
				if _, err := os.Stat(filepath.Join(pathMeta.User.Home, ".tool-versions")); !os.IsNotExist(err) {
					t.Errorf("Expected no global tool version file")
				}
				// Verify session removed
				stv, err := toolset.NewToolVersion(pathMeta.Working.SessionShim)
				if err != nil {
					t.Fatalf("Failed to read session tool version: %v", err)
				}
				if _, exists := stv.Record["test-sdk"]; exists {
					t.Errorf("Expected test-sdk to be removed from session tool version")
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing files
			os.RemoveAll(filepath.Join(pathMeta.User.Home, ".tool-versions"))
			os.RemoveAll(filepath.Join(pathMeta.Working.Directory, ".tool-versions"))
			os.RemoveAll(filepath.Join(pathMeta.Working.SessionShim, ".tool-versions"))

			// Setup
			tt.setupFunc()

			// Set hook environment to prevent shell reopen
			os.Setenv("__VFOX_SHELL", "1")
			defer os.Unsetenv("__VFOX_SHELL")

			// Execute
			err := sdk.Unuse(tt.scope)

			// Check error
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Verify
			if !tt.expectError {
				tt.verifyFunc(t)
			}
		})
	}
}
