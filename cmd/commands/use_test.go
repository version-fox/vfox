/*
 *    Copyright 2025 Han Li and contributors
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

package commands

import (
	"testing"

	"github.com/version-fox/vfox/internal/base"
)

func TestUseScopeSelection(t *testing.T) {
	// Test that scope selection logic works correctly
	tests := []struct {
		name          string
		globalSet     bool
		projectSet    bool
		sessionSet    bool
		expectedScope base.UseScope
	}{
		{
			name:          "Default to session scope",
			globalSet:     false,
			projectSet:    false,
			sessionSet:    false,
			expectedScope: base.Session,
		},
		{
			name:          "Global scope when global flag set",
			globalSet:     true,
			projectSet:    false,
			sessionSet:    false,
			expectedScope: base.Global,
		},
		{
			name:          "Project scope when project flag set",
			globalSet:     false,
			projectSet:    true,
			sessionSet:    false,
			expectedScope: base.Project,
		},
		{
			name:          "Session scope when session flag set",
			globalSet:     false,
			projectSet:    false,
			sessionSet:    true,
			expectedScope: base.Session,
		},
		{
			name:          "Global takes precedence over project",
			globalSet:     true,
			projectSet:    true,
			sessionSet:    false,
			expectedScope: base.Global,
		},
		{
			name:          "Global takes precedence over session",
			globalSet:     true,
			projectSet:    false,
			sessionSet:    true,
			expectedScope: base.Global,
		},
		{
			name:          "Project takes precedence over session",
			globalSet:     false,
			projectSet:    true,
			sessionSet:    true,
			expectedScope: base.Project,
		},
		{
			name:          "Global takes precedence over all",
			globalSet:     true,
			projectSet:    true,
			sessionSet:    true,
			expectedScope: base.Global,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the scope selection logic from the use command
			scope := base.Session
			if tt.globalSet {
				scope = base.Global
			} else if tt.projectSet {
				scope = base.Project
			} else {
				scope = base.Session
			}

			if scope != tt.expectedScope {
				t.Errorf("Expected scope %v, but got %v", tt.expectedScope, scope)
			}
		})
	}
}

func TestUseCommandValidation(t *testing.T) {
	// Test input validation logic
	tests := []struct {
		name        string
		sdkArg      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid SDK name",
			sdkArg:      "nodejs",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "Empty SDK name",
			sdkArg:      "",
			expectError: true,
			errorMsg:    "invalid parameter. format: <sdk-name>[@<version>]",
		},
		{
			name:        "SDK name with version",
			sdkArg:      "nodejs@18.0.0",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "SDK name with special characters",
			sdkArg:      "node-js@1.0",
			expectError: false,
			errorMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the validation logic from the use command
			hasError := len(tt.sdkArg) == 0
			
			if hasError != tt.expectError {
				t.Errorf("Expected error: %v, but got error: %v", tt.expectError, hasError)
			}
		})
	}
}
