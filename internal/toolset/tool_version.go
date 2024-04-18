package toolset

import (
	"fmt"
	"path/filepath"
)

const filename = ".tool-versions"

type MultiToolVersions []*ToolVersion

// FilterTools filters tools by the given filter function
// and return the first one you find.
func (m MultiToolVersions) FilterTools(filter func(name, version string) bool) map[string]string {
	tools := make(map[string]string)
	for _, t := range m {
		for name, version := range t.Record {
			_, ok := tools[name]
			if !ok && filter(name, version) {
				tools[name] = version
			}
		}
	}
	return tools
}

func (m MultiToolVersions) Add(name, version string) {
	for _, t := range m {
		t.Record[name] = version
	}
}

func (m MultiToolVersions) Save() error {
	for _, t := range m {
		if err := t.Save(); err != nil {
			return err
		}
	}
	return nil
}

// ToolVersion represents a .tool-versions file
type ToolVersion struct {
	*FileRecord
}

func NewToolVersion(dirPath string) (*ToolVersion, error) {
	file := filepath.Join(dirPath, filename)
	mapFile, err := NewFileRecord(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool versions file %s: %w", file, err)
	}
	return &ToolVersion{
		FileRecord: mapFile,
	}, nil
}

func NewMultiToolVersions(paths []string) (MultiToolVersions, error) {
	var tools MultiToolVersions
	for _, p := range paths {
		tool, err := NewToolVersion(p)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}
