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

import "strings"

const githubURL = "https://github.com"

type GitHub struct {
	BaseURL string `yaml:"baseUrl"`
}

var emptyGitHub = &GitHub{}

func (g *GitHub) WebBase() string {
	if g == nil || g.BaseURL == "" {
		return githubURL
	}
	return strings.TrimRight(g.BaseURL, "/")
}

func (g *GitHub) APIBase() string {
	base := g.WebBase()
	switch {
	case base == "https://github.com", base == "http://github.com",
		strings.HasSuffix(base, "/https://github.com"),
		strings.HasSuffix(base, "/http://github.com"):
		return strings.TrimSuffix(base, "github.com") + "api.github.com"
	default:
		return base
	}
}

func (g *GitHub) DownloadURL(source string) string {
	if path, ok := strings.CutPrefix(source, githubURL+"/"); ok {
		return g.WebBase() + "/" + path
	}
	return source
}
