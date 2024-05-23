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
	*util.SortedMap[string, string]
	path        string
	isInitEmpty bool
}

func (m *FileRecord) Save() error {
	if m.isInitEmpty && m.Len() == 0 {
		return nil
	}
	file, err := os.Create(m.path)
	if err != nil {
		return fmt.Errorf("failed to create file record %s: %w", m.path, err)
	}
	defer file.Close()

	err = m.ForEach(func(k string, v string) error {
		_, err = fmt.Fprintf(file, "%s %s\n", k, v)
		return err
	})
	if err != nil {
		return err
	}
	return nil
}

// NewFileRecord creates a new FileRecord from a file
// if the file does not exist, an empty FileRecord is returned
func NewFileRecord(path string) (*FileRecord, error) {
	versionsMap := util.NewSortedMap[string, string]()
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
				versionsMap.Set(parts[0], parts[1])
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	return &FileRecord{
		SortedMap:   versionsMap,
		path:        path,
		isInitEmpty: versionsMap.Len() == 0,
	}, nil
}
