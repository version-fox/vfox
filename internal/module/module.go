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

package module

import (
	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/module/html"
	"github.com/version-fox/vfox/internal/module/http"
	"github.com/version-fox/vfox/internal/module/json"
	"github.com/version-fox/vfox/internal/module/string"
	lua "github.com/yuin/gopher-lua"
)

func Preload(L *lua.LState, config *config.Config) {
	L.PreloadModule("http", http.NewModule(config.Proxy))
	json.Preload(L)
	html.Preload(L)
	string.Preload(L)
}
