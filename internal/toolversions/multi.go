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

package toolversions

type multi struct {
	main  Record
	slave []Record
}

func (m *multi) Export() map[string]string {
	return m.main.Export()
}

func (m *multi) Add(name, version string) error {
	err := m.main.Add(name, version)
	if err != nil {
		return err
	}
	for _, record := range m.slave {
		err = record.Add(name, version)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *multi) Close() error {
	err := m.main.Close()
	if err != nil {
		return err
	}
	for _, record := range m.slave {
		err = record.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
