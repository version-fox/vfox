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
	"fmt"
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

// LoadConfigByScope loads the config for the specified scope
// pathmeta is not aware of scope, but we don't need to track it in VfoxToml
func (m *RuntimeEnvContext) LoadConfigByScope(scope UseScope) (*pathmeta.VfoxToml, error) {
	var dir string

	switch scope {
	case Global:
		dir = m.PathMeta.User.Home
	case Project:
		dir = m.PathMeta.Working.Directory
	case Session:
		dir = m.PathMeta.Working.SessionShim
	default:
		return nil, fmt.Errorf("unknown scope: %v", scope)
	}

	return pathmeta.LoadConfig(dir)
}

// LoadConfigChainByScopes loads configs for multiple scopes and returns a chain
// Scopes are added in order (first added = lowest priority)
// Example: LoadConfigChainByScopes(Global, Session, Project) → Project has highest priority
func (m *RuntimeEnvContext) LoadConfigChainByScopes(scopes ...UseScope) (pathmeta.VfoxTomlChain, error) {
	chain := pathmeta.NewVfoxTomlChain()

	for _, scope := range scopes {
		c, err := m.LoadConfigByScope(scope)
		if err != nil {
			return chain, err
		}
		chain.Add(c)
	}

	return chain, nil
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
