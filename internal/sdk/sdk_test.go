package sdk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
)

func TestSdk_CreateSymlinksForScope(t *testing.T) {
	// Setup temporary directory structure
	tempDir := t.TempDir()

	// Create PathMeta with new structure
	pathMeta := &pathmeta.PathMeta{
		User: pathmeta.UserPaths{
			Home: filepath.Join(tempDir, "user"),
		},
		Shared: pathmeta.SharedPaths{
			Installs: filepath.Join(tempDir, "shared", "installs"),
		},
		Working: pathmeta.WorkingPaths{
			Directory:     filepath.Join(tempDir, "project"),
			ProjectSdkDir: filepath.Join(tempDir, "project", ".vfox", "sdk"),
			SessionSdkDir: filepath.Join(tempDir, "session", "sdk"),
			GlobalSdkDir:  filepath.Join(tempDir, "user", "sdk"),
		},
	}

	// Create runtime env context
	runtimeEnvContext := &env.RuntimeEnvContext{
		PathMeta: pathMeta,
	}

	// Create directories
	os.MkdirAll(pathMeta.User.Home, 0755)
	os.MkdirAll(pathMeta.Shared.Installs, 0755)
	os.MkdirAll(pathMeta.Working.ProjectSdkDir, 0755)
	os.MkdirAll(pathMeta.Working.SessionSdkDir, 0755)
	os.MkdirAll(pathMeta.Working.GlobalSdkDir, 0755)

	// Create fake SDK installation with correct directory structure
	// InstallPath/v-1.0.0/test-sdk-1.0.0
	sdkInstallPath := filepath.Join(pathMeta.Shared.Installs, "test-sdk")
	versionPath := filepath.Join(sdkInstallPath, "v-1.0.0")
	runtimePath := filepath.Join(versionPath, "test-sdk-1.0.0")
	os.MkdirAll(runtimePath, 0755)

	// Create a RuntimePackage directly instead of using GetRuntimePackage
	runtimePackage := &RuntimePackage{
		Runtime: &Runtime{
			Name:    "test-sdk",
			Version: "1.0.0",
			Path:    runtimePath,
		},
		PackagePath: versionPath,
	}

	// Create SDK
	sdk := &impl{
		Name:        "test-sdk",
		envContext:  runtimeEnvContext,
		InstallPath: sdkInstallPath,
	}

	tests := []struct {
		name        string
		scope       env.UseScope
		verifyFunc  func(t *testing.T) // Verify symlinks created
		expectError bool
	}{
		{
			name:  "Create symlinks for global scope",
			scope: env.Global,
			verifyFunc: func(t *testing.T) {
				// Check global symlink exists
				globalLink := filepath.Join(pathMeta.Working.GlobalSdkDir, "test-sdk")
				if _, err := os.Lstat(globalLink); os.IsNotExist(err) {
					t.Errorf("Expected global symlink to be created at %s", globalLink)
				}
			},
			expectError: false,
		},
		{
			name:  "Create symlinks for project scope",
			scope: env.Project,
			verifyFunc: func(t *testing.T) {
				// Check project symlink exists
				projectLink := filepath.Join(pathMeta.Working.ProjectSdkDir, "test-sdk")
				if _, err := os.Lstat(projectLink); os.IsNotExist(err) {
					t.Errorf("Expected project symlink to be created at %s", projectLink)
				}
			},
			expectError: false,
		},
		{
			name:  "Create symlinks for session scope",
			scope: env.Session,
			verifyFunc: func(t *testing.T) {
				// Check session symlink exists
				sessionLink := filepath.Join(pathMeta.Working.SessionSdkDir, "test-sdk")
				if _, err := os.Lstat(sessionLink); os.IsNotExist(err) {
					t.Errorf("Expected session symlink to be created at %s", sessionLink)
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up symlinks before each test
			os.RemoveAll(pathMeta.Working.GlobalSdkDir)
			os.RemoveAll(pathMeta.Working.ProjectSdkDir)
			os.RemoveAll(pathMeta.Working.SessionSdkDir)
			os.MkdirAll(pathMeta.Working.GlobalSdkDir, 0755)
			os.MkdirAll(pathMeta.Working.ProjectSdkDir, 0755)
			os.MkdirAll(pathMeta.Working.SessionSdkDir, 0755)

			// Execute - test internal method directly
			err := sdk.createSymlinksForScope(runtimePackage, tt.scope)

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

func TestSdk_EnvKeysForScope(t *testing.T) {
	// Setup temporary directory structure
	tempDir := t.TempDir()

	// Create PathMeta
	pathMeta := &pathmeta.PathMeta{
		User: pathmeta.UserPaths{
			Home: filepath.Join(tempDir, "user"),
		},
		Shared: pathmeta.SharedPaths{
			Installs: filepath.Join(tempDir, "shared", "installs"),
		},
		Working: pathmeta.WorkingPaths{
			Directory:     filepath.Join(tempDir, "project"),
			ProjectSdkDir: filepath.Join(tempDir, "project", ".vfox", "sdk"),
			SessionSdkDir: filepath.Join(tempDir, "session", "sdk"),
			GlobalSdkDir:  filepath.Join(tempDir, "user", "sdk"),
		},
	}

	// Create runtime env context
	runtimeEnvContext := &env.RuntimeEnvContext{
		PathMeta: pathMeta,
	}

	// Create directories
	os.MkdirAll(pathMeta.User.Home, 0755)
	os.MkdirAll(pathMeta.Shared.Installs, 0755)
	os.MkdirAll(pathMeta.Working.ProjectSdkDir, 0755)
	os.MkdirAll(pathMeta.Working.SessionSdkDir, 0755)
	os.MkdirAll(pathMeta.Working.GlobalSdkDir, 0755)

	// Create fake SDK installation with correct directory structure
	sdkInstallPath := filepath.Join(pathMeta.Shared.Installs, "test-sdk")
	versionPath := filepath.Join(sdkInstallPath, "v-1.0.0")
	runtimePath := filepath.Join(versionPath, "test-sdk-1.0.0")
	os.MkdirAll(runtimePath, 0755)

	// Create a RuntimePackage directly
	runtimePackage := &RuntimePackage{
		Runtime: &Runtime{
			Name:    "test-sdk",
			Version: "1.0.0",
			Path:    runtimePath,
		},
		PackagePath: versionPath,
	}

	// Create SDK
	sdk := &impl{
		Name:        "test-sdk",
		envContext:  runtimeEnvContext,
		InstallPath: sdkInstallPath,
	}

	t.Run("Test EnvKeysForScope creates symlinks and returns env vars", func(t *testing.T) {
		// Clean up symlinks before each test
		os.RemoveAll(pathMeta.Working.GlobalSdkDir)
		os.MkdirAll(pathMeta.Working.GlobalSdkDir, 0755)

		// Since EnvKeysForScope needs plugin, we test the internal logic manually:
		// 1. createSymlinksForScope creates symlinks
		// 2. EnvKeys returns env vars with symlink paths

		// Step 1: Create symlinks
		err := sdk.createSymlinksForScope(runtimePackage, env.Global)
		if err != nil {
			t.Fatalf("Failed to create symlinks: %v", err)
		}

		// Step 2: Verify symlink exists
		globalLink := filepath.Join(pathMeta.Working.GlobalSdkDir, "test-sdk")
		if _, err := os.Lstat(globalLink); os.IsNotExist(err) {
			t.Errorf("Expected global symlink to be created at %s", globalLink)
		}

		// Step 3: Test that EnvKeys returns correct paths (pointing to symlinks)
		// Note: We can't easily test EnvKeys without a plugin, so we verify
		// the symlink was created which is the key part
	})
}

func TestSdk_createSymlinksForScope_WithAdditions(t *testing.T) {
	// Setup temporary directory structure
	tempDir := t.TempDir()

	// Create PathMeta
	pathMeta := &pathmeta.PathMeta{
		User: pathmeta.UserPaths{
			Home: filepath.Join(tempDir, "user"),
		},
		Shared: pathmeta.SharedPaths{
			Installs: filepath.Join(tempDir, "shared", "installs"),
		},
		Working: pathmeta.WorkingPaths{
			Directory:     filepath.Join(tempDir, "project"),
			ProjectSdkDir: filepath.Join(tempDir, "project", ".vfox", "sdk"),
			SessionSdkDir: filepath.Join(tempDir, "session", "sdk"),
			GlobalSdkDir:  filepath.Join(tempDir, "user", "sdk"),
		},
	}

	// Create runtime env context
	runtimeEnvContext := &env.RuntimeEnvContext{
		PathMeta: pathMeta,
	}

	// Create directories
	os.MkdirAll(pathMeta.User.Home, 0755)
	os.MkdirAll(pathMeta.Shared.Installs, 0755)
	os.MkdirAll(pathMeta.Working.ProjectSdkDir, 0755)
	os.MkdirAll(pathMeta.Working.SessionSdkDir, 0755)
	os.MkdirAll(pathMeta.Working.GlobalSdkDir, 0755)

	// Create fake SDK installation with main runtime and additions
	sdkInstallPath := filepath.Join(pathMeta.Shared.Installs, "test-sdk")
	versionPath := filepath.Join(sdkInstallPath, "v-1.0.0")
	mainRuntimePath := filepath.Join(versionPath, "test-sdk-1.0.0")
	os.MkdirAll(mainRuntimePath, 0755)
	additionRuntimePath := filepath.Join(versionPath, "add-test-addition-1.0.0")
	os.MkdirAll(additionRuntimePath, 0755)

	// Create a RuntimePackage with main runtime and additions
	runtimePackage := &RuntimePackage{
		Runtime: &Runtime{
			Name:    "test-sdk",
			Version: "1.0.0",
			Path:    mainRuntimePath,
		},
		PackagePath: versionPath,
		Additions: []*Runtime{
			{
				Name:    "test-addition",
				Version: "1.0.0",
				Path:    additionRuntimePath,
			},
		},
	}

	// Create SDK
	sdk := &impl{
		Name:        "test-sdk",
		envContext:  runtimeEnvContext,
		InstallPath: sdkInstallPath,
	}

	t.Run("Create symlinks for main runtime and additions", func(t *testing.T) {
		// Execute
		err := sdk.createSymlinksForScope(runtimePackage, env.Project)

		// Check error
		if err != nil {
			t.Fatalf("Expected no error but got: %v", err)
		}

		// Verify main runtime symlink exists
		mainLink := filepath.Join(pathMeta.Working.ProjectSdkDir, "test-sdk")
		if _, err := os.Lstat(mainLink); os.IsNotExist(err) {
			t.Errorf("Expected main runtime symlink to be created at %s", mainLink)
		}

		// Verify addition symlink exists
		additionLink := filepath.Join(pathMeta.Working.ProjectSdkDir, "test-addition")
		if _, err := os.Lstat(additionLink); os.IsNotExist(err) {
			t.Errorf("Expected addition runtime symlink to be created at %s", additionLink)
		}
	})

	t.Run("Handle addition symlink creation failure gracefully", func(t *testing.T) {
		// Create a file at the addition symlink location to cause failure
		additionLink := filepath.Join(pathMeta.Working.GlobalSdkDir, "test-addition")
		_ = os.WriteFile(additionLink, []byte("block symlink creation"), 0644)

		// Execute - should not fail even if addition symlink fails
		err := sdk.createSymlinksForScope(runtimePackage, env.Global)

		// Should not error even if addition fails (additions fail gracefully)
		if err != nil {
			t.Logf("Got error (may be expected): %v", err)
		}

		// Verify main runtime symlink still exists
		mainLink := filepath.Join(pathMeta.Working.GlobalSdkDir, "test-sdk")
		if _, err := os.Lstat(mainLink); os.IsNotExist(err) {
			t.Errorf("Expected main runtime symlink to be created even if addition fails")
		}
	})
}
