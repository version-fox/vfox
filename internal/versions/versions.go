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

package versions

import (
	"bufio"
	"fmt"
	"github.com/version-fox/vfox/util"
	"os"
	"path/filepath"
	"strings"
)

const filename = ".tool-versions"

type ToolVersions struct {
	// tools sdkName -> version
	tools map[string]string
	path  string
}

func (t *ToolVersions) Save() error {
	file, err := os.Create(t.path)
	if err != nil {
		return err
	}
	defer file.Close()

	for k, v := range t.tools {
		_, err := fmt.Fprintf(file, "%s %s\n", k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *ToolVersions) Version(tool string) (string, bool) {
	s, ok := t.tools[tool]
	return s, ok
}

func (t *ToolVersions) Add(tool, version string) {
	t.tools[tool] = version
}

func NewToolVersions(dirPath string) (*ToolVersions, error) {
	versionsFile := filepath.Join(dirPath, filename)
	versionsMap := make(map[string]string)
	if util.FileExists(versionsFile) {
		file, err := os.Open(versionsFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, " ")
			if len(parts) == 2 {
				versionsMap[parts[0]] = parts[1]
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	return &ToolVersions{
		tools: versionsMap,
		path:  versionsFile,
	}, nil
}
