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

package env

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/shared/logger"
)

// RuntimeEnvContext represents the runtime environment context.
type RuntimeEnvContext struct {
	UserConfig        *config.Config     // UserConfig holds the user configuration.
	CurrentWorkingDir string             // CurrentWorkingDir is the current working directory.
	PathMeta          *pathmeta.PathMeta // PathMeta holds the path info of the environment.
	RuntimeVersion    string             // RuntimeVersion is the version of vfox
}

// LoadVfoxTomlByScope loads the config for the specified scope
// pathmeta is not aware of scope, but we don't need to track it in VfoxToml
func (m *RuntimeEnvContext) LoadVfoxTomlByScope(scope UseScope) (*pathmeta.VfoxToml, error) {
	var dir string

	switch scope {
	case Global:
		dir = m.PathMeta.User.Home
	case Project:
		dir = m.PathMeta.Working.Directory
	case Session:
		dir = m.PathMeta.Working.SessionSdkDir
	default:
		return nil, fmt.Errorf("unknown scope: %v", scope)
	}

	return pathmeta.LoadConfig(dir)
}

// LoadVfoxTomlChainByScopes loads configs for multiple scopes and returns a chain
// Scopes are added in order (first added = lowest priority)
// Example: LoadVfoxTomlChainByScopes(Global, Session, Project) → Project has highest priority
func (m *RuntimeEnvContext) LoadVfoxTomlChainByScopes(scopes ...UseScope) (VfoxTomlChain, error) {
	chain := NewVfoxTomlChain()

	for _, scope := range scopes {
		c, err := m.LoadVfoxTomlByScope(scope)
		if err != nil {
			return chain, err
		}
		chain.Add(c, scope)
	}

	return chain, nil
}

// HttpClient creates an HTTP client based on the proxy settings in the user configuration.
func (m *RuntimeEnvContext) HttpClient() *http.Client {
	var client *http.Client
	if m.UserConfig.Proxy.Enable {
		if uri, err := url.Parse(m.UserConfig.Proxy.Url); err == nil {
			transPort := &http.Transport{
				Proxy: http.ProxyURL(uri),
			}
			client = &http.Client{
				Transport: transPort,
			}
		}
	} else {
		client = http.DefaultClient
	}

	return client
}

// GetLinkDirPathByScope returns the symlink directory path for the given scope.
func (m *RuntimeEnvContext) GetLinkDirPathByScope(scope UseScope) string {
	var linkDir string
	switch scope {
	case Global:
		linkDir = m.PathMeta.Working.GlobalSdkDir
	case Project:
		linkDir = m.PathMeta.Working.ProjectSdkDir
	case Session:
		linkDir = m.PathMeta.Working.SessionSdkDir
	}
	return linkDir
}

// GetUserAddedPaths returns paths that the user has added to PATH after vfox activation.
// It compares the current PATH with the original PATH (stored during activation) and returns
// paths that are:
// 1. In the current PATH
// 2. NOT in the original PATH
// 3. NOT vfox-managed paths
func (m *RuntimeEnvContext) GetUserAddedPaths() *Paths {
	userPaths := NewPaths(EmptyPaths)

	// Get original PATH from environment (set during activation)
	originalPath := os.Getenv(OriginalPathFlag)
	if originalPath == "" {
		// No original path stored, return empty
		logger.Debugf("No original PATH found, skipping user path detection")
		return userPaths
	}

	// Parse original and current paths
	originalPaths := strings.Split(originalPath, string(os.PathListSeparator))
	currentPaths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))

	// Create a set of original paths for quick lookup
	originalPathSet := make(map[string]bool)
	for _, path := range originalPaths {
		if path != "" {
			originalPathSet[filepath.Clean(path)] = true
		}
	}

	// Find paths in current PATH that are NOT in original PATH and NOT vfox-managed
	for _, path := range currentPaths {
		if path == "" {
			continue
		}

		cleanPath := filepath.Clean(path)

		// Skip if it was in the original PATH
		if originalPathSet[cleanPath] {
			continue
		}

		// Skip if it's a vfox-managed path
		if pathmeta.IsVfoxRelatedPath(cleanPath) {
			continue
		}

		// This is a user-added path!
		logger.Debugf("Detected user-added path: %s", path)
		userPaths.Add(path)
	}

	return userPaths
}

// CleanSystemPaths returns system PATH with all vfox-managed paths removed (segment match).
// This ensures the system PATH is clean before adding vfox paths back in priority order.
func (m *RuntimeEnvContext) CleanSystemPaths() *Paths {
	// Get system paths
	systemPaths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))

	cleanPaths := NewPaths(EmptyPaths)
	for _, path := range systemPaths {
		cleanPath := filepath.Clean(path)

		if pathmeta.IsVfoxRelatedPath(cleanPath) {
			logger.Debugf("Removing vfox path from system PATH: %s", path)
			continue
		}
		cleanPaths.Add(path)
	}

	return cleanPaths
}
