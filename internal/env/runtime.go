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
	"net/http"
	"net/url"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/pathmeta"
)

// RuntimeEnvContext represents the runtime environment context.
type RuntimeEnvContext struct {
	UserConfig        *config.Config     // UserConfig holds the user configuration.
	CurrentWorkingDir string             // CurrentWorkingDir is the current working directory.
	PathMeta          *pathmeta.PathMeta // PathMeta holds the path info of the environment.
	RuntimeVersion    string             // RuntimeVersion is the version of vfox
}

// LoadToolVersionByScope loads the tool version based on the specified scope.
func (m *RuntimeEnvContext) LoadToolVersionByScope(scope UseScope) (*pathmeta.ToolVersion, error) {
	if Global == scope {
		return pathmeta.NewToolVersion(m.PathMeta.User.Home)
	} else if Project == scope {
		return pathmeta.NewToolVersion(m.PathMeta.Working.Directory)
	} else {
		return pathmeta.NewToolVersion(m.PathMeta.Working.SessionShim)
	}
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
