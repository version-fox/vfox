/*
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
 */

package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ConfigState tracks the state of configuration files for diff checking
type ConfigState struct {
	mu sync.RWMutex

	// Last check time (Unix timestamp in seconds)
	LastCheck int64 `json:"last_check"`

	// Config file modification times (Unix timestamps in seconds)
	GlobalMtime  int64 `json:"global_mtime,omitempty"`
	SessionMtime int64 `json:"session_mtime,omitempty"`
	ProjectMtime int64 `json:"project_mtime,omitempty"`

	// Cached env output (shell script)
	CachedOutput string `json:"cached_output,omitempty"`

	// State file path
	stateFilePath string
}

// NewConfigState creates a new ConfigState with the given state file path
func NewConfigState(stateFilePath string) *ConfigState {
	return &ConfigState{
		stateFilePath: stateFilePath,
		LastCheck:     0,
	}
}

// Load loads the state from disk
func (s *ConfigState) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// State file doesn't exist yet, that's ok
			return nil
		}
		return err
	}

	// Directly unmarshal into the state struct
	return json.Unmarshal(data, s)
}

// Save saves the state to disk
func (s *ConfigState) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

// saveLocked saves the state to disk (caller must hold lock)
func (s *ConfigState) saveLocked() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.stateFilePath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.stateFilePath, data, 0644)
}

// HasChanged checks if any config file has changed based on modification time
// Returns true if any config file has been modified since last check
func (s *ConfigState) HasChanged(configPaths map[UseScope]string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check each config file
	for scope, path := range configPaths {
		if path == "" {
			continue
		}

		mtime, err := getFileModTime(path)
		if err != nil {
			// File doesn't exist or can't be read
			if os.IsNotExist(err) {
				// File was deleted, consider it as changed
				return true, nil
			}
			return false, err
		}

		// Compare with stored mtime
		var storedMtime int64
		switch scope {
		case Global:
			storedMtime = s.GlobalMtime
		case Session:
			storedMtime = s.SessionMtime
		case Project:
			storedMtime = s.ProjectMtime
		}

		if mtime > storedMtime {
			// File has been modified
			return true, nil
		}
	}

	return false, nil
}

// Update updates the state with new config mtimes and cached output
func (s *ConfigState) Update(configPaths map[UseScope]string, output string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LastCheck = time.Now().Unix()

	// Update mtimes for all config files
	for scope, path := range configPaths {
		if path == "" {
			continue
		}

		mtime, err := getFileModTime(path)
		if err != nil {
			// File doesn't exist, skip it
			continue
		}

		switch scope {
		case Global:
			s.GlobalMtime = mtime
		case Session:
			s.SessionMtime = mtime
		case Project:
			s.ProjectMtime = mtime
		}
	}

	s.CachedOutput = output

	return s.saveLocked()
}

// GetCachedOutput returns the cached env output
func (s *ConfigState) GetCachedOutput() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CachedOutput
}

// getFileModTime returns the modification time of a file as Unix timestamp
func getFileModTime(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.ModTime().Unix(), nil
}
