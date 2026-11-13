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
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserAgent(t *testing.T) {
	tests := []struct {
		name                                      string
		runtimeVersion, pluginName, pluginVersion string
		want                                      string
	}{
		{"vfox only", "1.0.13", "", "", "vfox/1.0.13"},
		{"plugin", "1.0.13", "gradle", "0.0.3", "vfox/1.0.13 vfox-gradle/0.0.3"},
		{"plugin name already prefixed", "1.0.13", "vfox-gradle", "0.0.3", "vfox/1.0.13 vfox-gradle/0.0.3"},
		{"no plugin version", "1.0.13", "gradle", "", "vfox/1.0.13 vfox-gradle"},
		{"no runtime version", "", "gradle", "0.0.3", "vfox vfox-gradle/0.0.3"},
		{"nothing", "", "", "", "vfox"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UserAgent(tt.runtimeVersion, tt.pluginName, tt.pluginVersion); got != tt.want {
				t.Fatalf("UserAgent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWithDefaultUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.UserAgent()))
	}))
	defer srv.Close()
	client := &http.Client{Transport: WithDefaultUserAgent(nil, "vfox/1.0.13")}

	tests := []struct {
		name   string
		header map[string]string
		want   string
	}{
		{"no agent of its own", nil, "vfox/1.0.13"},
		{"own agent", map[string]string{"User-Agent": "custom"}, "custom"},
		{"own agent in lower case", map[string]string{"user-agent": "custom"}, "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			for key, value := range tt.header {
				req.Header.Add(key, value)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Do() error = %v", err)
			}
			defer resp.Body.Close()
			got := make([]byte, 64)
			n, _ := resp.Body.Read(got)
			if string(got[:n]) != tt.want {
				t.Fatalf("server saw User-Agent %q, want %q", got[:n], tt.want)
			}
			if tt.header == nil && req.Header.Get("User-Agent") != "" {
				t.Fatalf("the caller's request was modified: %v", req.Header)
			}
		})
	}
}
