package toolset

import (
	"bufio"
	"fmt"
	"github.com/version-fox/vfox/internal/util"
	"os"
	"path/filepath"
	"strings"
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

type ToolVersion struct {
	// Sdks sdkName -> version
	Record map[string]string
	path   string
}

func (t *ToolVersion) Save() error {
	if len(t.Record) == 0 {
		return nil
	}
	file, err := os.Create(t.path)
	if err != nil {
		return err
	}
	defer file.Close()

	for k, v := range t.Record {
		_, err := fmt.Fprintf(file, "%s %s\n", k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewToolVersion(dirPath string) (*ToolVersion, error) {
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
	return &ToolVersion{
		Record: versionsMap,
		path:   file,
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
