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

package internal

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/version-fox/vfox/internal/plugin"
)

type PluginRunOptions struct {
	OS, Arch    string
	Offline     bool
	Timeout     time.Duration
	Diagnostics io.Writer
}

type PluginTestOptions struct {
	File                string
	Online              bool
	Timeout             time.Duration
	Output, Diagnostics io.Writer
}

func pluginDevelopmentDirectory(directory string) (string, error) {
	path, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("plugin path must be a directory: %s", path)
	}
	return path, nil
}

// RunPluginHook deliberately bypasses NewSdkManager and all user/project state.
func RunPluginHook(ctx context.Context, directory, hook string, input []byte, options PluginRunOptions) (any, error) {
	directory, err := pluginDevelopmentDirectory(directory)
	if err != nil {
		return nil, err
	}
	if options.Timeout == 0 {
		options.Timeout = 60 * time.Second
	}
	if options.Timeout < 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}
	ctx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	return plugin.RunDevelopmentHook(ctx, directory, hook, input, plugin.DevelopmentOptions{
		RuntimeVersion: RuntimeVersion, OS: options.OS, Arch: options.Arch,
		Online: !options.Offline, Output: options.Diagnostics,
	})
}

// TestPlugin discovers files and reports each result immediately. Each file gets
// a fresh VM and deadline, including when a previous file failed or timed out.
func TestPlugin(ctx context.Context, directory string, options PluginTestOptions) error {
	directory, err := pluginDevelopmentDirectory(directory)
	if err != nil {
		return err
	}
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}
	if options.Timeout < 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if options.Output == nil {
		options.Output = os.Stdout
	}
	if options.Diagnostics == nil {
		options.Diagnostics = os.Stderr
	}
	var files []string
	if options.File != "" {
		file := options.File
		if !filepath.IsAbs(file) {
			file = filepath.Join(directory, file)
		}
		info, err := os.Stat(file)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("test path must be a file: %s", file)
		}
		files = append(files, file)
	} else {
		tests := filepath.Join(directory, "tests")
		err := filepath.WalkDir(tests, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), "_test.lua") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if len(files) == 0 {
		return fmt.Errorf("no test files found in %s", filepath.Join(directory, "tests"))
	}
	sort.Strings(files)
	failed := 0
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		fileCtx, cancel := context.WithTimeout(ctx, options.Timeout)
		err := plugin.TestDevelopmentFile(fileCtx, directory, file, plugin.DevelopmentOptions{
			RuntimeVersion: RuntimeVersion, Online: options.Online, Output: options.Diagnostics,
		})
		cancel()
		label, relErr := filepath.Rel(directory, file)
		if relErr != nil {
			label = file
		}
		label = filepath.ToSlash(label)
		if err != nil {
			failed++
			if _, writeErr := fmt.Fprintf(options.Output, "FAIL %s\n", label); writeErr != nil {
				return writeErr
			}
			if _, writeErr := fmt.Fprintf(options.Diagnostics, "%s: %s\n", label, err); writeErr != nil {
				return writeErr
			}
		} else if _, err := fmt.Fprintf(options.Output, "PASS %s\n", label); err != nil {
			return err
		}
	}
	if failed != 0 {
		return fmt.Errorf("%d of %d test files failed", failed, len(files))
	}
	return nil
}
