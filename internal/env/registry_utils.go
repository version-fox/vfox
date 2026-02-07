package env

import (
	"path/filepath"
	"strings"
)

func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	cleaned := filepath.Clean(path)
	return strings.ToLower(cleaned)
}

func dedupOrderedPaths(paths []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		norm := normalizePath(trimmed)
		if norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func removePaths(existing, toRemove []string) []string {
	if len(existing) == 0 {
		return nil
	}
	removeSet := make(map[string]struct{}, len(toRemove))
	for _, path := range toRemove {
		if path == "" {
			continue
		}
		if norm := normalizePath(path); norm != "" {
			removeSet[norm] = struct{}{}
		}
	}
	result := make([]string, 0, len(existing))
	for _, path := range existing {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		if norm := normalizePath(trimmed); norm != "" {
			if _, ok := removeSet[norm]; ok {
				continue
			}
		}
		result = append(result, trimmed)
	}
	return result
}

func splitSemicolonSeparated(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ";")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
