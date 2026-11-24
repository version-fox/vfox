/*
 *
 *    Copyright 2026 Han Li and contributors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 *
 */

package pathmeta

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shared/util"
)

type UserPaths struct {
	Home   string // ~/.vfox
	Temp   string // ~/.vfox/tmp (session temporary)
	Config string // ~/.vfox/config.yaml (user override config)
}

type SharedPaths struct {
	Root     string // /opt/vfox or C:\Program Files\vfox (from VFOX_ROOT or default)
	Installs string // ${SharedRoot}/installs (SDK unique install location)
	Plugins  string // ${SharedRoot}/plugins (plugins)
	Config   string // ${SharedRoot}/config.yaml (global config)
}

type WorkingPaths struct {
	Directory     string // Current working directory
	ProjectSdkDir string // .vfox/sdk (project-level)
	SessionSdkDir string // Session temporary shim path
	GlobalSdkDir  string // Global shim path in user home
}

type PathMeta struct {
	User       UserPaths
	Shared     SharedPaths
	Working    WorkingPaths
	Executable string      // Executable path
	InitErrors []InitError // Best-effort directory/path initialization failures
}

// InitError records a non-fatal directory or path resolution failure encountered
// while constructing PathMeta so diagnostic commands (`doctor`, `use`) can report it
// without polluting read paths (`env`, `activate`).
type InitError struct {
	Target string
	Path   string
	Err    error
}

const (
	HookCurTmpPath = "__VFOX_CURTMPPATH"

	oldVfoxDirPrefix    = ".version-fox" // Old vfox dir prefix
	vfoxDirPrefix       = ".vfox"
	pluginDirPrefix     = "plugin" // Shared plugins directory
	tmpDirPrefix        = "tmp"    // User-level temp
	installedDirPrefix  = "cache"  // Shared installed directory
	configFilePrefix    = "config.yaml"
	symlinkSdkDirPrefix = "sdks"

	ReadWriteAuth = 0755 // can write and read
)

// IsVfoxRelatedPath checks if the given path is related to vfox
func IsVfoxRelatedPath(path string) bool {
	cleanPath := filepath.Clean(path)
	parts := []string{oldVfoxDirPrefix, vfoxDirPrefix}

	for _, part := range parts {
		if containsPathSegment(cleanPath, part) {
			return true
		}
	}
	return false
}

func containsPathSegment(path string, part string) bool {
	// Normalize path to use system separator
	// This handles both Unix-style (/) and Windows-style (\) paths
	normalizedPath := filepath.FromSlash(path)

	// Split path by separator to check individual segments
	separator := string(filepath.Separator)
	segments := strings.Split(normalizedPath, separator)
	for _, segment := range segments {
		if segment == part {
			return true
		}
	}
	return false
}

func newTempPath(pid int) string {
	now := time.Now()
	dateStr := now.Format("20060102")
	name := fmt.Sprintf("%s-%d", dateStr, pid)
	return name
}

// NewPathMeta creates a new PathMeta instance based on the provided parameters and environment variables.
// If not provide sharedRoot, it will use userHome as sharedRoot.
func NewPathMeta(userHome, sharedRoot, currentDir string, pid int) *PathMeta {
	vfoxUserHome := filepath.Join(userHome, oldVfoxDirPrefix)
	// Compatibility check for old directory
	if !util.FileExists(vfoxUserHome) {
		vfoxUserHome = filepath.Join(userHome, vfoxDirPrefix)
	}
	// Use userHome as sharedRoot if not provided
	if len(sharedRoot) == 0 {
		sharedRoot = vfoxUserHome
	}
	exePath, exeErr := getExecutablePath()
	meta := &PathMeta{
		User: UserPaths{
			Home:   vfoxUserHome,
			Temp:   filepath.Join(vfoxUserHome, tmpDirPrefix),
			Config: filepath.Join(vfoxUserHome, configFilePrefix),
		},
		Shared: SharedPaths{
			Root:     sharedRoot,
			Installs: filepath.Join(sharedRoot, installedDirPrefix),
			Plugins:  filepath.Join(sharedRoot, pluginDirPrefix),
			Config:   filepath.Join(sharedRoot, configFilePrefix),
		},
		Working: WorkingPaths{
			Directory:     currentDir,
			ProjectSdkDir: filepath.Join(currentDir, vfoxDirPrefix, symlinkSdkDirPrefix),
			SessionSdkDir: generateSessionShimPath(filepath.Join(vfoxUserHome, tmpDirPrefix), pid),
			GlobalSdkDir:  filepath.Join(vfoxUserHome, symlinkSdkDirPrefix),
		},
		Executable: exePath,
	}
	if exeErr != nil {
		meta.InitErrors = append(meta.InitErrors, InitError{
			Target: "executable path",
			Err:    exeErr,
		})
	}

	// Initialize user directories and immediate shared subdirectories
	// best-effort (Shared subdirs are created only when Shared.Root already
	// exists; write commands ensure missing parents).
	ensureUserTempDirs(meta)
	if err := os.MkdirAll(meta.Working.GlobalSdkDir, ReadWriteAuth); err != nil {
		logger.Debugf("create global sdk dir failed: %v\n", err)
		meta.InitErrors = append(meta.InitErrors, InitError{
			Target: "global sdk directory",
			Path:   meta.Working.GlobalSdkDir,
			Err:    err,
		})
	}
	if util.FileExists(meta.Shared.Root) {
		for _, dir := range []string{meta.Shared.Installs, meta.Shared.Plugins} {
			if err := os.Mkdir(dir, ReadWriteAuth); err != nil && !errors.Is(err, fs.ErrExist) {
				logger.Debugf("create shared directory %s failed: %v\n", dir, err)
				meta.InitErrors = append(meta.InitErrors, InitError{
					Target: "shared directory",
					Path:   dir,
					Err:    err,
				})
			}
		}
	}

	return meta
}

func isWithinDir(parent, target string) bool {
	rel, err := filepath.Rel(parent, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func ensureUserTempDirs(meta *PathMeta) {
	// Ensure User.Temp in one place: when HookCurTmpPath is unset, SessionSdkDir
	// lives directly under User.Temp so a single MkdirAll prepares both; when set,
	// fail at most once if SessionSdkDir shares the same blocked User.Temp root.
	tempReady := true
	if os.Getenv(HookCurTmpPath) != "" {
		if err := os.MkdirAll(meta.User.Temp, ReadWriteAuth); err != nil {
			logger.Debugf("create temp dir failed: %v\n", err)
			meta.InitErrors = append(meta.InitErrors, InitError{
				Target: "user temp directory",
				Path:   meta.User.Temp,
				Err:    err,
			})
			tempReady = false
		}
	}
	if !util.FileExists(meta.Working.SessionSdkDir) {
		if !tempReady && isWithinDir(meta.User.Temp, meta.Working.SessionSdkDir) {
			return
		}
		if err := os.MkdirAll(meta.Working.SessionSdkDir, ReadWriteAuth); err != nil {
			logger.Debugf("create temp dir failed: %v\n", err)
			meta.InitErrors = append(meta.InitErrors, InitError{
				Target: "temp directory",
				Path:   meta.Working.SessionSdkDir,
				Err:    err,
			})
		}
	}
}

func generateSessionShimPath(userTemp string, pid int) string {
	curTmpPath := os.Getenv(HookCurTmpPath)
	if curTmpPath == "" {
		name := newTempPath(pid)
		curTmpPath = filepath.Join(userTemp, name)
	}
	return curTmpPath
}

// Executable resolves the running binary path; tests override it to simulate lookup failures.
var Executable = os.Executable

func getExecutablePath() (string, error) {
	exePath, err := Executable()
	if err != nil {
		logger.Debugf("get executable path failed: %v\n", err)
		return "", err
	}
	return exePath, nil
}

// ApplyStoragePath applies the storage path from config to update the Installs path
// If storagePath is empty, the current Installs path is unchanged
func (p *PathMeta) ApplyStoragePath(storagePath string) error {
	if storagePath == "" {
		return nil
	}

	// Update the Installs path to use the configured storage path
	p.Shared.Installs = filepath.Join(storagePath)

	// Ensure the directory exists
	if err := os.MkdirAll(p.Shared.Installs, ReadWriteAuth); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	return nil
}
