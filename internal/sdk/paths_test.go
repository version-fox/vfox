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
 *
 */

package sdk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/plugin"
)

func TestReportedPathOfAnInstalledVersion(t *testing.T) {
	var seen *plugin.PathsHookCtx
	p := (&testPlugin{paths: func(ctx *plugin.PathsHookCtx) (*plugin.PathsHookResult, error) {
		seen = ctx
		return &plugin.PathsHookResult{Paths: map[string]string{
			"include": filepath.Join(ctx.Path, "include"),
			"lib":     filepath.Join(ctx.Path, "lib"),
		}}, nil
	}}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")
	inst, err := s.Installation("1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"include", "lib"} {
		if err := os.MkdirAll(filepath.Join(inst.Runtime.Path, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.ReportedPath("1.0.0", "include")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(inst.Runtime.Path, "include"); got != want {
		t.Errorf("ReportedPath = %q, want %q", got, want)
	}
	if seen == nil || seen.Path != inst.Runtime.Path || seen.Version != "1.0.0" {
		t.Errorf("hook context = %+v, want the main runtime directory of 1.0.0", seen)
	}
	// A name the plugin does not report is no error, just nothing.
	if got, err := s.ReportedPath("1.0.0", "bin"); err != nil || got != "" {
		t.Errorf("ReportedPath(bin) = %q, %v, want nothing", got, err)
	}
}

func TestReportedPathRejectsADirectoryThatIsNotThere(t *testing.T) {
	for name, value := range map[string]func(main string) string{
		"relative": func(string) string { return "include" },
		"missing":  func(main string) string { return filepath.Join(main, "missing") },
	} {
		t.Run(name, func(t *testing.T) {
			p := (&testPlugin{paths: func(ctx *plugin.PathsHookCtx) (*plugin.PathsHookResult, error) {
				return &plugin.PathsHookResult{Paths: map[string]string{"include": value(ctx.Path)}}, nil
			}}).withCommand(t, "tool")
			s := newTestSDK(t, p)
			mustInstall(t, s, "1.0.0")
			if got, err := s.ReportedPath("1.0.0", "include"); err == nil {
				t.Errorf("ReportedPath = %q, want an error", got)
			}
		})
	}
}

func TestReportedPathWhenThePluginReportsNothing(t *testing.T) {
	// No Paths hook at all.
	s := newTestSDK(t, (&testPlugin{}).withCommand(t, "tool"))
	mustInstall(t, s, "1.0.0")
	if got, err := s.ReportedPath("1.0.0", "include"); err != nil || got != "" {
		t.Errorf("without the hook: %q, %v", got, err)
	}

	// The hook is there and has nothing to say.
	s = newTestSDK(t, (&testPlugin{paths: func(*plugin.PathsHookCtx) (*plugin.PathsHookResult, error) {
		return nil, plugin.ErrNoResultProvide
	}}).withCommand(t, "tool"))
	mustInstall(t, s, "1.0.0")
	if got, err := s.ReportedPath("1.0.0", "include"); err != nil || got != "" {
		t.Errorf("with nothing to say: %q, %v", got, err)
	}
}

func TestReportedPathOfAnUninstalledVersion(t *testing.T) {
	s := newTestSDK(t, &testPlugin{paths: func(*plugin.PathsHookCtx) (*plugin.PathsHookResult, error) {
		t.Error("the hook must not run for a version that is not installed")
		return nil, plugin.ErrNoResultProvide
	}})
	if _, err := s.ReportedPath("9.9.9", "include"); err == nil {
		t.Error("ReportedPath of a version that is not installed must fail")
	}
}
