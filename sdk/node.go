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

package sdk

import (
	"fmt"
	"github.com/aooohan/version-fox/env"
	"github.com/aooohan/version-fox/util"
	"net/url"
	"path/filepath"
)

const nodeDownloadUrl = "https://nodejs.org/dist/v%s/node-v%s-%s-%s%s"

type NodeSource string

func (n NodeSource) DownloadUrl(handler *Handler, version Version) *url.URL {
	archType := handler.Operation.ArchType
	if archType == "amd64" {
		archType = "x64"
	}
	urlStr := fmt.Sprintf(nodeDownloadUrl, version, version, handler.Operation.OsType, archType, n.FileExt(handler))
	u, _ := url.Parse(urlStr)
	return u
}

func (n NodeSource) FileExt(handler *Handler) string {
	if handler.Operation.OsType == util.Windows {
		return ".zip"
	}
	return ".tar.gz"
}

func (n NodeSource) EnvKeys(handler *Handler, version Version) []*env.KV {
	return []*env.KV{
		{
			"PATH",
			filepath.Join(handler.VersionPath(version), "bin"),
		},
	}
}

func (n NodeSource) Name() string {
	return "node"
}

func NewNodeSource() NodeSource {
	return NodeSource("node")
}
