/*
 *    Copyright 2024 Han Li and contributors
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
	"github.com/version-fox/vfox/internal/config"
	"testing"
)

func TestNewConfig(t *testing.T) {
	_, err := config.NewConfig("")
	if err != nil {
		t.Fatal(err)
	}
}

func TestConfig_Proxy(t *testing.T) {
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
}
func TestConfigWithEmptyProxy(t *testing.T) {
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
}
