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

package config

import "testing"

func TestGitHubAPIBase(t *testing.T) {
	tests := []struct {
		name   string
		github *GitHub
		want   string
	}{
		{name: "nil receiver", github: nil, want: "https://api.github.com"},
		{name: "empty base", github: &GitHub{}, want: "https://api.github.com"},
		{name: "https github.com", github: &GitHub{BaseURL: "https://github.com"}, want: "https://api.github.com"},
		{name: "http github.com", github: &GitHub{BaseURL: "http://github.com"}, want: "http://api.github.com"},
		{name: "trailing slash", github: &GitHub{BaseURL: "https://github.com/"}, want: "https://api.github.com"},
		{name: "https path proxy", github: &GitHub{BaseURL: "https://gh-proxy.org/https://github.com"}, want: "https://gh-proxy.org/https://api.github.com"},
		{name: "https path proxy trailing slash", github: &GitHub{BaseURL: "https://gh-proxy.org/https://github.com/"}, want: "https://gh-proxy.org/https://api.github.com"},
		{name: "http path proxy", github: &GitHub{BaseURL: "https://gh-proxy.org/http://github.com"}, want: "https://gh-proxy.org/http://api.github.com"},
		{name: "replace trailing github.com only", github: &GitHub{BaseURL: "https://github.com.example/https://github.com"}, want: "https://github.com.example/https://api.github.com"},
		{name: "host mirror prefix", github: &GitHub{BaseURL: "https://kkgithub.com"}, want: "https://kkgithub.com"},
		{name: "host mirror trailing slash", github: &GitHub{BaseURL: "https://kkgithub.com/"}, want: "https://kkgithub.com"},
		{name: "host mirror suffix", github: &GitHub{BaseURL: "https://github.com.example"}, want: "https://github.com.example"},
		{name: "non-url path suffix", github: &GitHub{BaseURL: "https://mirror.example.com/github.com"}, want: "https://mirror.example.com/github.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.github.APIBase(); got != tc.want {
				t.Errorf("APIBase() = %q, want %q", got, tc.want)
			}
		})
	}
}
