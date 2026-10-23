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

import "fmt"

const aliasesKey = "aliases"

// AliasFile is the aliases of one config file. Saving writes the aliases and
// nothing else, as File does for any key: the file keeps its other keys and its
// comments, and a section it does not declare stays undeclared.
type AliasFile struct {
	// Aliases are the ones the file has, alias by SDK name: Aliases[sdk][alias]
	// is a version. Change them, then Save.
	Aliases map[string]map[string]string

	file *File
}

// LoadAliasFile reads the aliases of the config file at path. A file that does
// not exist has none, and Save creates it.
func LoadAliasFile(path string) (*AliasFile, error) {
	file, err := LoadFile(path)
	if err != nil {
		return nil, err
	}
	f := &AliasFile{file: file, Aliases: make(map[string]map[string]string)}

	var decoded struct {
		Aliases map[string]map[string]string `yaml:"aliases"`
	}
	if err := file.root.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("read the aliases of %s: %w", path, err)
	}
	if decoded.Aliases != nil {
		f.Aliases = decoded.Aliases
	}
	return f, nil
}

// Save writes the aliases back to the file. An SDK without aliases is not
// kept, and the key goes when no alias is left.
func (f *AliasFile) Save() error {
	for sdk, names := range f.Aliases {
		if len(names) == 0 {
			delete(f.Aliases, sdk)
		}
	}
	if len(f.Aliases) == 0 {
		f.file.remove([]string{aliasesKey})
	} else if err := f.file.put([]string{aliasesKey}, f.Aliases); err != nil {
		return err
	}
	return f.file.Save()
}
