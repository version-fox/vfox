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
	Record map[string]string
	Path   string
}

func (m *FileRecord) Save() error {
	if len(m.Record) == 0 {
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
		Record: versionsMap,
		Path:   path,
	}, nil
}
