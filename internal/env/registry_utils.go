package env

import (
	"path"
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

// ReplaceInstallRootPrefix rewrites value when oldRoot is a full path segment
// of it. A semicolon-separated value, such as Path, is rewritten one entry at a
// time. Matching is case-insensitive, the way normalizePath is. A value that
// merely contains the old text as a substring is left as it is. Slash and
// backslash are both separators, so a Windows registry value matches on every
// OS. A UNC destination such as \\server\share\sdks keeps its prefix.
func ReplaceInstallRootPrefix(value, oldRoot, newRoot string) string {
	if strings.TrimSpace(oldRoot) == "" || value == "" {
		return value
	}
	parts := strings.Split(value, ";")
	for i, part := range parts {
		parts[i] = replaceInstallRootPart(part, oldRoot, newRoot)
	}
	return strings.Join(parts, ";")
}

func replaceInstallRootPart(part, oldRoot, newRoot string) string {
	trimmed := strings.TrimSpace(part)
	if trimmed == "" {
		return part
	}
	swapped, ok := swapInstallRootPrefix(trimmed, oldRoot, newRoot)
	if !ok {
		return part
	}
	left := len(part) - len(strings.TrimLeft(part, " \t"))
	right := len(part) - len(strings.TrimRight(part, " \t"))
	return part[:left] + swapped + part[len(part)-right:]
}

func swapInstallRootPrefix(value, oldRoot, newRoot string) (string, bool) {
	valueParts := slashParts(value)
	oldParts := slashParts(oldRoot)
	if len(oldParts) == 0 || len(valueParts) < len(oldParts) {
		return value, false
	}
	for i := range oldParts {
		if !strings.EqualFold(valueParts[i], oldParts[i]) {
			return value, false
		}
	}
	rest := valueParts[len(oldParts):]
	out := append(slashParts(newRoot), rest...)
	return withSeparator(strings.Join(out, "/"), separatorOf(value)), true
}

func slashParts(p string) []string {
	p = strings.ReplaceAll(p, `\`, `/`)
	p = cleanKeepingUNC(p)
	if p == "/" {
		return []string{""}
	}
	if p == "." || p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// cleanKeepingUNC cleans a Windows path written with slashes. filepath.Clean
// on Windows leaves a UNC volume unchanged; path.Clean collapses //server/share
// to /server/share. The volume is split off first, the way that cleaner does,
// and only the path under it is cleaned.
func cleanKeepingUNC(p string) string {
	vol, rest := splitUNCVolume(p)
	if vol == "" {
		return path.Clean(p)
	}
	if rest == "" {
		return vol
	}
	rest = path.Clean(rest)
	if rest == "." {
		return vol
	}
	if strings.HasPrefix(rest, "/") {
		return vol + rest
	}
	return vol + "/" + rest
}

// splitUNCVolume returns the //host/share volume of a slash path and the
// remainder under it. A path that is not UNC returns "", p.
func splitUNCVolume(p string) (string, string) {
	if len(p) < 2 || p[0] != '/' || p[1] != '/' || (len(p) > 2 && p[2] == '/') {
		return "", p
	}
	rest := strings.TrimLeft(p[2:], "/")
	host, rest, ok := cutSlash(rest)
	if host == "" {
		return "", p
	}
	if !ok {
		return "//" + host, ""
	}
	share, rest, ok := cutSlash(rest)
	if share == "" {
		return "//" + host, ""
	}
	vol := "//" + host + "/" + share
	if !ok || rest == "" {
		return vol, ""
	}
	return vol, "/" + rest
}

func cutSlash(p string) (before, after string, found bool) {
	i := strings.IndexByte(p, '/')
	if i < 0 {
		return p, "", false
	}
	return p[:i], strings.TrimLeft(p[i+1:], "/"), true
}

func separatorOf(p string) string {
	if strings.Contains(p, `\`) && !strings.Contains(p, `/`) {
		return `\`
	}
	return `/`
}

func withSeparator(p, sep string) string {
	if sep == `\` {
		return strings.ReplaceAll(p, `/`, `\`)
	}
	return strings.ReplaceAll(p, `\`, `/`)
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
