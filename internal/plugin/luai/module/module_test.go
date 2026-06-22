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

package module

import (
	"testing"

	"github.com/version-fox/vfox/internal/config"
	lua "github.com/yuin/gopher-lua"
)

func TestPreloadIncludesFileModule(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	Preload(L, &PreloadOptions{Config: config.DefaultConfig})
	if err := L.DoString(`
		local fs = require("fs")
		assert(type(fs.copy) == "function")
		assert(type(fs.remove) == "function")
		assert(type(fs.move) == "function")
		assert(type(fs.symlink) == "function")
	`); err != nil {
		t.Fatal(err)
	}
}
