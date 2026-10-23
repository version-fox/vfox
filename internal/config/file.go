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
	"strings"

	"gopkg.in/yaml.v3"
)

// File is a config file that vfox edits, read and written as a YAML document
// and not as a Config. Changing a key changes that key and nothing else: the
// file keeps its other keys and its comments, and a section it does not declare
// stays undeclared, so that it keeps following the shared config. Loading the
// file as a Config and saving that would write every default and every shared
// value into it as if the user had chosen them, and a section the file declares
// replaces the shared one.
type File struct {
	path string
	doc  yaml.Node
	root *yaml.Node // the mapping at the top of doc
	// comments is what a file holds when it has comments and nothing else: the
	// parser keeps no node for them, so they are written back as they were.
	comments []byte
}

// LoadFile reads the config file at path. A file that does not exist is an
// empty one, and Save creates it.
func LoadFile(path string) (*File, error) {
	f := &File{path: path}
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
	return f, nil
}

// Save writes the file. A file that vfox could not load is not written: it
// stays as it was and the error says why.
func (f *File) Save() error {
	content, err := yaml.Marshal(&f.doc)
	if err != nil {
		return err
	}
	if _, err := decodeConfig(content); err != nil {
		return fmt.Errorf("%s would not load after this change, so it is left as it is: %w", f.path, err)
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
func (f *File) topMapping() (*yaml.Node, error) {
	if f.doc.Kind == 0 {
		f.doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	if f.doc.Kind != yaml.DocumentNode || len(f.doc.Content) != 1 {
		return nil, errors.New("not a YAML document")
	}
	root := f.doc.Content[0]
	if isNull(root) {
		*root = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", HeadComment: root.HeadComment, FootComment: root.FootComment}
	}
	if root.Kind != yaml.MappingNode {
		return nil, errors.New("the top level of the file is not a mapping")
	}
	return root, nil
}

// put sets the key at path to value, which is encoded as YAML. The sections on
// the way are created when the file lacks them; nothing else changes. A value
// that replaces a scalar keeps the comments around the old one.
func (f *File) put(path []string, value any) error {
	node := &yaml.Node{}
	if err := node.Encode(value); err != nil {
		return err
	}

	mapping := f.root
	for i, name := range path[:len(path)-1] {
		section, _ := entry(mapping, name)
		switch {
		case section == nil:
			section = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			appendEntry(mapping, name, section)
		case isNull(section): // a key with nothing under it
			*section = yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", HeadComment: section.HeadComment, LineComment: section.LineComment, FootComment: section.FootComment}
		case section.Kind != yaml.MappingNode:
			return fmt.Errorf("%s: %s is not a section", f.path, strings.Join(path[:i+1], "."))
		}
		mapping = section
	}

	name := path[len(path)-1]
	old, _ := entry(mapping, name)
	switch {
	case old == nil:
		appendEntry(mapping, name, node)
	case old.Kind == yaml.ScalarNode && node.Kind == yaml.ScalarNode:
		node.HeadComment, node.LineComment, node.FootComment = old.HeadComment, old.LineComment, old.FootComment
		*old = *node
	default:
		*old = *node
	}
	return nil
}

// remove takes the key at path out of the file, and the sections that this
// leaves empty: a section without a key is still one the file declares, and it
// would replace the shared one. It reports whether the file had the key.
func (f *File) remove(path []string) bool {
	holders := make([]*yaml.Node, 0, len(path)) // holders[i] is the mapping that holds path[i]
	mapping := f.root
	for i, name := range path {
		holders = append(holders, mapping)
		next, _ := entry(mapping, name)
		if next == nil {
			return false
		}
		if i < len(path)-1 {
			if next.Kind != yaml.MappingNode {
				return false
			}
			mapping = next
		}
	}
	for i := len(path) - 1; i >= 0; i-- {
		_, index := entry(holders[i], path[i])
		holders[i].Content = append(holders[i].Content[:index], holders[i].Content[index+2:]...)
		if len(holders[i].Content) > 0 {
			break
		}
	}
	return true
}

// entry returns the value of a key in a mapping and the index of the key.
func entry(mapping *yaml.Node, name string) (*yaml.Node, int) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if key := mapping.Content[i]; key.Kind == yaml.ScalarNode && key.Value == name {
			return mapping.Content[i+1], i
		}
	}
	return nil, -1
}

// appendEntry adds a key to the end of a mapping. A mapping that is empty (`{}`)
// takes its first key as a block.
func appendEntry(mapping *yaml.Node, name string, value *yaml.Node) {
	if len(mapping.Content) == 0 {
		mapping.Style &^= yaml.FlowStyle
	}
	key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}
	mapping.Content = append(mapping.Content, key, value)
}

func isNull(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode && node.Tag == "!!null"
}
