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

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/shared/cache"
)

func TestNewConfig(t *testing.T) {
	c, err := config.NewConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if c.Proxy.Url != "http://test" {
		t.Fatal("proxy url is invalid")
	}
	if !c.Proxy.Enable == false {
		t.Fatal("proxy enable is invalid")
	}
	if c.Storage.SdkPath != "/tmp" {
		t.Fatal("storage sdk path is invalid")
	}
	if !c.LegacyVersionFile.Enable {
		t.Fatal("legacy version file enable is invalid")
	}
	if c.Cache.AvailableHookDuration != cache.Duration(-1) {
		t.Fatal("cache available hook duration is invalid")
	}
}

func TestConfigWithEmpty(t *testing.T) {
	c, err := config.NewConfigWithPath("empty_test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if c.Proxy.Url != "" {
		t.Fatal("proxy url must be empty")
	}
	if !c.Proxy.Enable == false {
		t.Fatal("proxy enable must be false")
	}
	if c.Storage.SdkPath != "" {
		t.Fatal("proxy url must be empty")
	}
	if c.LegacyVersionFile.Enable != true {
		t.Fatal("legacy version file enable must be true")
	}
	if c.Registry.Address != "" {
		t.Fatal("registry address must be empty")
	}
}

func TestStorageWithWritePermission(t *testing.T) {
	dir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	s := &config.Storage{
		SdkPath: dir,
	}
	if err = s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestStorageValidateKeepsExistingFiles(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, ".tmpfile")
	if err := os.WriteFile(existing, []byte("user data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := (&config.Storage{SdkPath: dir}).Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(existing)
	if err != nil || string(data) != "user data" {
		t.Fatalf("existing file changed: %q, %v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected only the existing file, got %v, %v", entries, err)
	}
}

func TestGitHubMirrorInheritance(t *testing.T) {
	dir := t.TempDir()
	shared, user := filepath.Join(dir, "shared.yaml"), filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(shared, []byte("github:\n  baseUrl: https://shared.example\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"https://shared.example", "https://user.example", "https://shared.example"} {
		cfg, err := config.LoadConfigWithFallback(shared, user)
		if err != nil || cfg.GitHub.WebBase() != want {
			t.Fatalf("expected %s, config %v, error %v", want, cfg, err)
		}
		f, err := config.LoadFile(user)
		if err != nil {
			t.Fatal(err)
		}
		if want == "https://shared.example" {
			err = f.Set("github.baseUrl", "https://user.example")
		} else {
			_, err = f.Unset("github.baseUrl")
		}
		if err != nil {
			t.Fatal(err)
		}
		if err = f.Save(); err != nil {
			t.Fatal(err)
		}
	}
	g := config.DefaultConfig.GitHub
	if g.WebBase() != "https://github.com" || g.APIBase() != "https://api.github.com" {
		t.Fatal("unexpected GitHub defaults")
	}
	const thirdParty = "https://github.com.example/plugin.zip"
	if got := (&config.GitHub{BaseURL: "https://mirror.example"}).DownloadURL(thirdParty); got != thirdParty {
		t.Fatalf("rewrote %s", got)
	}
}
