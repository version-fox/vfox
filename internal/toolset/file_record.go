/*
 *    Copyright 2024 Han Li and contributors
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

package toolset

import (
	"bufio"
	"fmt"
	"github.com/version-fox/vfox/internal/util"
	"os"
	"strings"
)

// FileRecord is a file that contains a map of string to string
type FileRecord struct {
	Record      map[string]string
	Path        string
	isInitEmpty bool
}

func (m *FileRecord) Save() error {
	if m.isInitEmpty && len(m.Record) == 0 {
		return nil
	}
	file, err := os.Create(m.Path)
	if err != nil {
		return fmt.Errorf("failed to create file record %s: %w", m.Path, err)
	}
	defer file.Close()

	for k, v := range m.Record {
		_, err := fmt.Fprintf(file, "%s %s\n", k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// NewFileRecord creates a new FileRecord from a file
// if the file does not exist, an empty FileRecord is returned
func NewFileRecord(path string) (*FileRecord, error) {
	versionsMap := make(map[string]string)
	if util.FileExists(path) {
		file, err := os.Open(path)
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
	return &FileRecord{
		Record:      versionsMap,
		Path:        path,
		isInitEmpty: len(versionsMap) == 0,
	}, nil
}
