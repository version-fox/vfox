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

package util

import (
	"fmt"
	"net/http"
	"strings"
)

// UserAgent returns the User-Agent of a request made by vfox:
// "vfox/<runtimeVersion>", followed by "vfox-<pluginName>/<pluginVersion>"
// when the request is made for a plugin. A version that is empty is left out,
// and so is its slash.
func UserAgent(runtimeVersion, pluginName, pluginVersion string) string {
	components := make([]string, 0, 2)
	if runtimeVersion != "" {
		components = append(components, fmt.Sprintf("vfox/%s", runtimeVersion))
	} else {
		components = append(components, "vfox")
	}

	if pluginName != "" {
		if !strings.HasPrefix(pluginName, "vfox-") {
			pluginName = "vfox-" + pluginName
		}
		if pluginVersion != "" {
			components = append(components, fmt.Sprintf("%s/%s", pluginName, pluginVersion))
		} else {
			components = append(components, pluginName)
		}
	}

	return strings.TrimSpace(strings.Join(components, " "))
}

// WithDefaultUserAgent returns a transport that sends userAgent with every
// request that carries no User-Agent of its own. Without it such a request
// goes out as "Go-http-client", which some mirrors refuse.
func WithDefaultUserAgent(base http.RoundTripper, userAgent string) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &userAgentTransport{base: base, userAgent: userAgent}
}

type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		// A RoundTripper must not modify the request it is given.
		req = req.Clone(req.Context())
		req.Header.Set("User-Agent", t.userAgent)
	}
	return t.base.RoundTrip(req)
}
