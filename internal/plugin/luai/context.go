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

package luai

import (
	"fmt"
	"strings"
	"sync"
)

// Context stores runtime and plugin metadata shared by Lua modules.
type Context struct {
	mu             sync.RWMutex
	runtimeVersion string
	pluginName     string
	pluginVersion  string
	userAgent      string
}

func NewContext(runtimeVersion string) *Context {
	ctx := &Context{
		runtimeVersion: runtimeVersion,
	}
	ctx.composeUserAgent()
	return ctx
}

// RuntimeVersion returns the runtime version.
func (c *Context) RuntimeVersion() string {
	if c == nil {
		return ""
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.runtimeVersion
}

// PluginInfo returns the plugin name and version.
func (c *Context) PluginInfo() (string, string) {
	if c == nil {
		return "", ""
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.pluginName, c.pluginVersion
}

// SetPluginInfo updates the plugin name and version stored in the context.
func (c *Context) SetPluginInfo(name, version string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.pluginName = name
	c.pluginVersion = version
	c.composeUserAgent()
}

// UserAgent returns the composed default User-Agent string.
func (c *Context) UserAgent() string {
	if c == nil {
		return ""
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userAgent
}

func (c *Context) composeUserAgent() {
	components := make([]string, 0, 2)
	if c.runtimeVersion != "" {
		components = append(components, fmt.Sprintf("vfox/%s", c.runtimeVersion))
	} else {
		components = append(components, "vfox")
	}

	name := ensurePrefix(c.pluginName)
	if name != "" {
		if c.pluginVersion != "" {
			components = append(components, fmt.Sprintf("%s/%s", name, c.pluginVersion))
		} else {
			components = append(components, name)
		}
	}

	c.userAgent = strings.TrimSpace(strings.Join(components, " "))
}

func ensurePrefix(name string) string {
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "vfox-") {
		return name
	}
	return "vfox-" + name
}
