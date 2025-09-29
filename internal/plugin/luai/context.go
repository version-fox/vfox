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
)

// computeUserAgent constructs a user agent string for the vfox runtime and plugin.
// 
// Parameters:
//   - runtimeVersion: the version of the vfox runtime (may be empty).
//   - pluginName: the name of the plugin (will be prefixed with "vfox-" if not already).
//   - pluginVersion: the version of the plugin (may be empty).
//
// Returns:
//   A user agent string in the format "vfox/<runtimeVersion> vfox-<pluginName>/<pluginVersion>",
//   omitting version information if not provided, and trimming extra spaces.
func computeUserAgent(runtimeVersion, pluginName, pluginVersion string) string {
	components := make([]string, 0, 2)
	if runtimeVersion != "" {
		components = append(components, fmt.Sprintf("vfox/%s", runtimeVersion))
	} else {
		components = append(components, "vfox")
	}

	name := ensurePrefix(pluginName)
	if name != "" {
		if pluginVersion != "" {
			components = append(components, fmt.Sprintf("%s/%s", name, pluginVersion))
		} else {
			components = append(components, name)
		}
	}

	return strings.TrimSpace(strings.Join(components, " "))
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
