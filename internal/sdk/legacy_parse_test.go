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

package sdk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/plugin"
)

func TestSdkParseLegacyFile_UsesDeclaredLegacyFilenames(t *testing.T) {
	envContext := &env.RuntimeEnvContext{
		UserConfig:     config.DefaultConfig,
		RuntimeVersion: "test",
	}

	plug, err := plugin.CreatePlugin(filepath.Join("..", "plugin", "testdata", "plugins", "java_with_metadata"), envContext)
	if err != nil {
		t.Fatalf("create plugin: %v", err)
	}
	defer plug.Close()

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".node-version"), []byte("14.17.0\n"), 0644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	sdk := &impl{
		Name:       "java_with_metadata",
		envContext: envContext,
		plugin:     plug,
	}

	version, err := sdk.ParseLegacyFile(projectDir)
	if err != nil {
		t.Fatalf("parse legacy file: %v", err)
	}
	if version != "14.17.0" {
		t.Fatalf("expected legacy file version 14.17.0, got %q", version)
	}
}
