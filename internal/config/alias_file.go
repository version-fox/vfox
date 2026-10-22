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

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

const aliasesKey = "aliases"

// AliasFile is the aliases of one config file. Saving writes the aliases and
// nothing else: the file keeps its other keys and its comments, and a section it
// does not declare stays undeclared, so that it keeps following the shared
// config. Loading the file as a Config and saving that would write every
// default into it as if the user had chosen it.
type AliasFile struct {
	// Aliases are the ones the file has, alias by SDK name: Aliases[sdk][alias]
	// is a version. Change them, then Save.
	Aliases map[string]map[string]string

	path string
	doc  yaml.Node
	root *yaml.Node // the mapping at the top of doc
	// comments is what a file holds when it has comments and nothing else: the
	// parser keeps no node for them, so they are written back as they were.
	comments []byte
}

// LoadAliasFile reads the aliases of the config file at path. A file that does
// not exist has none, and Save creates it.
func LoadAliasFile(path string) (*AliasFile, error) {
	f := &AliasFile{path: path, Aliases: make(map[string]map[string]string)}
	content, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := yaml.Unmarshal(content, &f.doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if f.doc.Kind == 0 && len(bytes.TrimSpace(content)) > 0 {
			f.comments = content
		}
	case !errors.Is(err, fs.ErrNotExist):
		return nil, err
	}
	if f.root, err = f.topMapping(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	var decoded struct {
		Aliases map[string]map[string]string `yaml:"aliases"`
	}
	if err := f.root.Decode(&decoded); err != nil {
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
	if err := setAliases(f.root, f.Aliases); err != nil {
		return err
	}

	content, err := yaml.Marshal(&f.doc)
	if err != nil {
		return err
	}
	var out []byte
	if len(f.comments) > 0 {
		out = append(out, f.comments...)
		if !bytes.HasSuffix(out, []byte("\n")) {
			out = append(out, '\n')
		}
	}
	return os.WriteFile(f.path, append(out, content...), 0666)
}

// topMapping returns the mapping at the top of the document. A file without
// content gets an empty one, and so does a file that holds only a null.
func (f *AliasFile) topMapping() (*yaml.Node, error) {
	if f.doc.Kind == 0 {
		f.doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	if f.doc.Kind != yaml.DocumentNode || len(f.doc.Content) != 1 {
		return nil, errors.New("not a YAML document")
	}
	root := f.doc.Content[0]
	if root.Kind == yaml.ScalarNode && root.Tag == "!!null" {
		*root = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", HeadComment: root.HeadComment, FootComment: root.FootComment}
	}
	if root.Kind != yaml.MappingNode {
		return nil, errors.New("the top level of the file is not a mapping")
	}
	return root, nil
}

// setAliases puts the aliases under the aliases key of root, or takes the key
// out when there are none. Nothing else in root is touched.
func setAliases(root *yaml.Node, aliases map[string]map[string]string) error {
	index := -1
	for i := 0; i+1 < len(root.Content); i += 2 {
		if key := root.Content[i]; key.Kind == yaml.ScalarNode && key.Value == aliasesKey {
			index = i
			break
		}
	}
	if len(aliases) == 0 {
		if index >= 0 {
			root.Content = append(root.Content[:index], root.Content[index+2:]...)
		}
		return nil
	}

	value := &yaml.Node{}
	if err := value.Encode(aliases); err != nil {
		return err
	}
	if index >= 0 {
		root.Content[index+1] = value
		return nil
	}
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: aliasesKey}
	root.Content = append(root.Content, key, value)
	return nil
}
