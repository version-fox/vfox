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
	"io"
	nethttp "net/http"

	lua "github.com/yuin/gopher-lua"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/plugin/luai/module/archiver"
	"github.com/version-fox/vfox/internal/plugin/luai/module/fs"
	"github.com/version-fox/vfox/internal/plugin/luai/module/html"
	"github.com/version-fox/vfox/internal/plugin/luai/module/http"
	"github.com/version-fox/vfox/internal/plugin/luai/module/json"
	"github.com/version-fox/vfox/internal/plugin/luai/module/string"
)

type PreloadOptions struct {
	Config        *config.Config
	HTTPTransport func(nethttp.RoundTripper) nethttp.RoundTripper
	Output        io.Writer
}

func Preload(L *lua.LState, options *PreloadOptions) {
	cfg := options.Config
	if cfg == nil {
		cfg = config.DefaultConfig
	}
	http.PreloadWithOptions(L, cfg.Proxy, cfg.Plugin.HTTP, http.Options{
		WrapTransport: options.HTTPTransport,
		Output:        options.Output,
	})
	json.Preload(L)
	html.Preload(L)
	string.Preload(L)
	archiver.Preload(L)
	fs.Preload(L, "")
}
