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
	"testing"

	"github.com/version-fox/vfox/internal/config"
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
	if c.Cache.AvailableHookDuration != config.CacheDuration(-1) {
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
