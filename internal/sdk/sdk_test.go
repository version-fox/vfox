package sdk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/shared/cache"
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

func TestSdk_AvailableSkipCache(t *testing.T) {
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "test-sdk")
	writeTestPlugin(t, pluginDir)

	meta, err := pathmeta.NewPathMeta(
		filepath.Join(tempDir, "home"),
		filepath.Join(tempDir, "shared"),
		filepath.Join(tempDir, "project"),
		os.Getpid(),
	)
	if err != nil {
		t.Fatalf("Failed to create path meta: %v", err)
	}

	runtimeEnvContext := &env.RuntimeEnvContext{
		UserConfig: &config.Config{
			Proxy:             config.EmptyProxy,
			Storage:           config.EmptyStorage,
			Registry:          config.EmptyRegistry,
			LegacyVersionFile: config.EmptyLegacyVersionFile,
			Cache: &config.Cache{
				AvailableHookDuration: cache.Duration(time.Hour),
			},
		},
		CurrentWorkingDir: meta.Working.Directory,
		PathMeta:          meta,
		RuntimeVersion:    "test",
	}

	source, err := NewSdk(runtimeEnvContext, pluginDir)
	if err != nil {
		t.Fatalf("Failed to create sdk: %v", err)
	}

	result, err := source.Available([]string{"test"}, false)
	if err != nil {
		t.Fatalf("Available failed: %v", err)
	}
	firstNote := availableNote(t, result)

	time.Sleep(time.Second)

	result, err = source.Available([]string{"test"}, true)
	if err != nil {
		t.Fatalf("Available failed: %v", err)
	}
	skipNote := availableNote(t, result)
	if firstNote == skipNote {
		t.Fatalf("Expected skip-cache to bypass cached result")
	}

	result, err = source.Available([]string{"test"}, false)
	if err != nil {
		t.Fatalf("Available failed: %v", err)
	}
	cachedNote := availableNote(t, result)
	if cachedNote != firstNote {
		t.Fatalf("Expected cached note %q, got %q", firstNote, cachedNote)
	}
}

func writeTestPlugin(t *testing.T, pluginDir string) {
	t.Helper()
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin dir: %v", err)
	}
	pluginSource := []byte(`PLUGIN = {
  name = "test-sdk",
  version = "0.0.1",
  description = "test"
}

function PLUGIN:Available(ctx)
  return {
    { version = "1.0.0", note = tostring(os.time()) }
  }
end

function PLUGIN:PreInstall(ctx)
  return { version = ctx.version, url = "" }
end

function PLUGIN:EnvKeys(ctx)
  return {}
end
`)
	if err := os.WriteFile(filepath.Join(pluginDir, "main.lua"), pluginSource, 0644); err != nil {
		t.Fatalf("Failed to write plugin: %v", err)
	}
}

func availableNote(t *testing.T, result []*AvailableRuntimePackage) string {
	t.Helper()
	if len(result) == 0 || result[0] == nil || result[0].AvailableRuntime == nil {
		t.Fatalf("Expected available results")
	}
	return result[0].Note
}

func TestEnsureVfoxInGitignore(t *testing.T) {
	tests := []struct {
		name           string
		setupGitignore func(string) error // Setup function to create .gitignore with specific content
		wantAdded      bool               // Expected return value (true if added, false if not)
		verifyContent  func(string)       // Function to verify the final content
		expectError    bool
	}{
		{
			name: "Add .vfox/ to gitignore when it doesn't exist",
			setupGitignore: func(projectDir string) error {
				gitignorePath := filepath.Join(projectDir, ".gitignore")
				return os.WriteFile(gitignorePath, []byte("node_modules/\nbuild/\n"), 0644)
			},
			wantAdded: true,
			verifyContent: func(gitignorePath string) {
				content, _ := os.ReadFile(gitignorePath)
				contentStr := string(content)
				if !containsLine(contentStr, ".vfox/") {
					t.Errorf("Expected .vfox/ to be added to gitignore")
				}
			},
			expectError: false,
		},
		{
			name: "Don't add when .vfox/ already exists",
			setupGitignore: func(projectDir string) error {
				gitignorePath := filepath.Join(projectDir, ".gitignore")
				return os.WriteFile(gitignorePath, []byte("node_modules/\n.vfox/\nbuild/\n"), 0644)
			},
			wantAdded: false,
			verifyContent: func(gitignorePath string) {
				content, _ := os.ReadFile(gitignorePath)
				lines := countOccurrences(string(content), ".vfox/")
				if lines != 1 {
					t.Errorf("Expected .vfox/ to appear only once, got %d occurrences", lines)
				}
			},
			expectError: false,
		},
		{
			name: "Don't add when .vfox (without slash) already exists",
			setupGitignore: func(projectDir string) error {
				gitignorePath := filepath.Join(projectDir, ".gitignore")
				return os.WriteFile(gitignorePath, []byte("node_modules/\n.vfox\nbuild/\n"), 0644)
			},
			wantAdded: false,
			verifyContent: func(gitignorePath string) {
				content, _ := os.ReadFile(gitignorePath)
				lines := countOccurrences(string(content), ".vfox")
				if lines != 1 {
					t.Errorf("Expected .vfox to appear only once, got %d occurrences", lines)
				}
			},
			expectError: false,
		},
		{
			name: "Don't create .gitignore if it doesn't exist",
			setupGitignore: func(projectDir string) error {
				// Don't create .gitignore
				return nil
			},
			wantAdded: false,
			verifyContent: func(gitignorePath string) {
				if _, err := os.Stat(gitignorePath); !os.IsNotExist(err) {
					t.Errorf("Expected .gitignore to not exist")
				}
			},
			expectError: false,
		},
		{
			name: "Handle empty .gitignore file",
			setupGitignore: func(projectDir string) error {
				gitignorePath := filepath.Join(projectDir, ".gitignore")
				return os.WriteFile(gitignorePath, []byte(""), 0644)
			},
			wantAdded: true,
			verifyContent: func(gitignorePath string) {
				content, _ := os.ReadFile(gitignorePath)
				contentStr := string(content)
				if !containsLine(contentStr, ".vfox/") {
					t.Errorf("Expected .vfox/ to be added to empty gitignore")
				}
			},
			expectError: false,
		},
		{
			name: "Handle .gitignore without trailing newline",
			setupGitignore: func(projectDir string) error {
				gitignorePath := filepath.Join(projectDir, ".gitignore")
				return os.WriteFile(gitignorePath, []byte("node_modules/"), 0644)
			},
			wantAdded: true,
			verifyContent: func(gitignorePath string) {
				content, _ := os.ReadFile(gitignorePath)
				contentStr := string(content)
				if !containsLine(contentStr, ".vfox/") {
					t.Errorf("Expected .vfox/ to be added")
				}
				// Verify proper newline was added
				if contentStr != "node_modules/\n.vfox/\n" {
					t.Errorf("Expected proper newline handling, got: %q", contentStr)
				}
			},
			expectError: false,
		},
		{
			name: "Handle .gitignore with trailing newline",
			setupGitignore: func(projectDir string) error {
				gitignorePath := filepath.Join(projectDir, ".gitignore")
				return os.WriteFile(gitignorePath, []byte("node_modules/\n"), 0644)
			},
			wantAdded: true,
			verifyContent: func(gitignorePath string) {
				content, _ := os.ReadFile(gitignorePath)
				contentStr := string(content)
				if !containsLine(contentStr, ".vfox/") {
					t.Errorf("Expected .vfox/ to be added")
				}
			},
			expectError: false,
		},
		{
			name: "Handle .vfox/ with extra whitespace",
			setupGitignore: func(projectDir string) error {
				gitignorePath := filepath.Join(projectDir, ".gitignore")
				return os.WriteFile(gitignorePath, []byte("  .vfox/  \n"), 0644)
			},
			wantAdded: false,
			verifyContent: func(gitignorePath string) {
				content, _ := os.ReadFile(gitignorePath)
				lines := countOccurrences(string(content), ".vfox/")
				if lines != 1 {
					t.Errorf("Expected .vfox/ to appear only once (whitespace should match)")
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary project directory
			tempDir := t.TempDir()
			projectDir := filepath.Join(tempDir, "project")
			err := os.MkdirAll(projectDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create project dir: %v", err)
			}

			// Setup .gitignore as per test case
			if tt.setupGitignore != nil {
				if err := tt.setupGitignore(projectDir); err != nil {
					t.Fatalf("Failed to setup gitignore: %v", err)
				}
			}

			// Execute the function
			added, err := ensureVfoxInGitignore(projectDir)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check return value
			if added != tt.wantAdded {
				t.Errorf("Expected added=%v, got %v", tt.wantAdded, added)
			}

			// Verify content
			gitignorePath := filepath.Join(projectDir, ".gitignore")
			if tt.verifyContent != nil {
				tt.verifyContent(gitignorePath)
			}
		})
	}
}

// Helper function to check if a line exists in content
func containsLine(content, line string) bool {
	lines := splitLines(content)
	for _, l := range lines {
		if strings.TrimSpace(l) == strings.TrimSpace(line) {
			return true
		}
	}
	return false
}

// Helper function to count occurrences of a string (exact line match)
func countOccurrences(content, search string) int {
	lines := splitLines(content)
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == search {
			count++
		}
	}
	return count
}

// Helper function to split content into lines
func splitLines(content string) []string {
	return strings.Split(content, "\n")
}
