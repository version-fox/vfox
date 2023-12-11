/*
 *    Copyright 2023 [lihan aooohan@gmail.com]
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

package plugin

import (
	"github.com/aooohan/version-fox/env"
	"github.com/aooohan/version-fox/util"
	"net/url"
)

type Plugin interface {
	DownloadUrl(ctx *Context) *url.URL
	EnvKeys(ctx *Context) []*env.KV
	Search(ctx *Context) []SearchResult
	Name() string
	Close()
}

type Context struct {
	util.OSType
	util.ArchType
	Version string
}

type SearchResult string
