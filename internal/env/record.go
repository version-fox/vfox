/*
 *    Copyright 2024 [lihan aooohan@gmail.com]
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

package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/version-fox/vfox/internal/util"
)

const filename = ".tool-versions"

func IsRecordExist(dirPath string) bool {
	return util.FileExists(filepath.Join(dirPath, filename))
}

// Record is an interface to record tool version
type Record interface {
	Add(name, version string)
	Export() map[string]string
	Save() error
}
type empty struct {
}

func (e empty) Add(name, version string) {
}

func (e empty) Export() map[string]string {
	return map[string]string{}
}

func (e empty) Save() error {
	return nil
}

var EmptyRecord = &empty{}

type single struct {
	// Sdks sdkName -> version
	Sdks map[string]string
	path string
}

func (t *single) Export() map[string]string {
	return t.Sdks
}

func (t *single) Save() error {
	if len(t.Sdks) == 0 {
		return nil
	}
	file, err := os.Create(t.path)
	if err != nil {
		return err
	}
	defer file.Close()

	for k, v := range t.Sdks {
		_, err := fmt.Fprintf(file, "%s %s\n", k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *single) String() string {
	return filename
}

func (t *single) Add(name, version string) {
	t.Sdks[name] = version
	return
}

func newSingle(dirPath string) (Record, error) {
	file := filepath.Join(dirPath, filename)
	versionsMap := make(map[string]string)
	if util.FileExists(file) {
		file, err := os.Open(file)
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
	return &single{
		Sdks: versionsMap,
		path: file,
	}, nil
}

type multi struct {
	main  Record
	slave []Record
}

func (m *multi) Export() map[string]string {
	result := make(map[string]string)
	for k, v := range m.main.Export() {
		result[k] = v
	}
	for _, s := range m.slave {
		for k, v := range s.Export() {
			result[k] = v
		}
	}
	return result
}

func (m *multi) Add(name, version string) {
	m.main.Add(name, version)
	for _, record := range m.slave {
		record.Add(name, version)
	}
}

func (m *multi) Save() error {
	err := m.main.Save()
	if err != nil {
		return err
	}
	for _, record := range m.slave {
		err = record.Save()
		if err != nil {
			return err
		}
	}
	return nil
}

func NewRecord(mainPath string, salve ...string) (Record, error) {
	main, err := newSingle(mainPath)
	if err != nil {
		return nil, fmt.Errorf("read version record failed, error: %w", err)
	}

	if len(salve) == 0 {
		return main, nil
	}

	var salveRecords []Record
	for _, path := range salve {
		if path == "" {
			continue
		}
		salveRecord, err := newSingle(path)
		if err != nil {
			return nil, fmt.Errorf("read version record failed, error: %w", err)
		}
		salveRecords = append(salveRecords, salveRecord)
	}
	return &multi{
		main:  main,
		slave: salveRecords,
	}, nil
}
