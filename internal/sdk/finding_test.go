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

package sdk

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/pterm/pterm"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
)

// capturePterm collects what pterm prints to its default output, which is where
// `vfox use` talks to the user.
func capturePterm(t *testing.T) *bytes.Buffer {
	t.Helper()
	var out bytes.Buffer
	pterm.SetDefaultOutput(&out)
	t.Cleanup(func() { pterm.SetDefaultOutput(os.Stdout) })
	return &out
}

func TestApplyIgnoreNeedsAFunctionToDoAnything(t *testing.T) {
	findings := []Finding{{ID: "a", SDK: "x", Version: "1"}}
	ApplyIgnore(findings, nil)
	if findings[0].Ignored {
		t.Error("without an ignore function nothing is ignored")
	}
	ApplyIgnore(findings, func(id, sdk, version string) bool { return id == "a" && sdk == "x" && version == "1" })
	if !findings[0].Ignored {
		t.Error("the ignore function was not asked")
	}
}

// Findings that agree on SDK, version and severity are told apart by ID and
// then by message, so the order does not depend on how the checks ran.
func TestSortFindingsBreaksTiesByIDAndMessage(t *testing.T) {
	findings := []Finding{
		{ID: "b", Severity: SeverityWarn, SDK: "s", Version: "1", Message: "second"},
		{ID: "a", Severity: SeverityWarn, SDK: "s", Version: "1", Message: "z"},
		{ID: "b", Severity: SeverityWarn, SDK: "s", Version: "1", Message: "first"},
		{ID: "a", Severity: SeverityWarn, SDK: "s", Version: "1", Message: "a"},
	}
	SortFindings(findings)
	var got []string
	for _, f := range findings {
		got = append(got, f.ID+":"+f.Message)
	}
	if want := []string{"a:a", "a:z", "b:first", "b:second"}; !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestPrintFindingsTalksToPterm(t *testing.T) {
	out := capturePterm(t)
	PrintFindings([]Finding{
		{ID: "a", Severity: SeverityError, Message: "broken thing", Fix: []string{"vfox fix it"}},
		{ID: "b", Severity: SeverityWarn, Message: "odd thing", Hints: []string{"because of this"}},
		{ID: "c", Severity: SeverityInfo, Message: "only a notice"},
	}, SeverityWarn)

	got := out.String()
	for _, want := range []string{"broken thing", "vfox fix it", "odd thing", "because of this"} {
		if !strings.Contains(got, want) {
			t.Errorf("output lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "only a notice") {
		t.Errorf("a notice is below the level that is printed:\n%s", got)
	}
}

// `vfox use` calls reportFindings for the version it just selected.
func TestReportFindings(t *testing.T) {
	t.Run("says what shadows the version", func(t *testing.T) {
		s := installedSDK(t)
		f := newPathFixture(t, "pnpm")
		out := capturePterm(t)

		s.reportFindings("1.0.0", true)
		got := out.String()
		if !strings.Contains(got, "comes before vfox in PATH") || !strings.Contains(got, f.shadow) {
			t.Errorf("output = %q, want the warning that names the shadowing executable", got)
		}
	})

	t.Run("does not look at PATH when the shell is not going to run the version", func(t *testing.T) {
		s := installedSDK(t)
		newPathFixture(t, "pnpm")
		out := capturePterm(t)

		s.reportFindings("1.0.0", false)
		if out.Len() != 0 {
			t.Errorf("output = %q, want nothing without a hook in the shell", out.String())
		}
	})

	t.Run("can be switched off", func(t *testing.T) {
		s := installedSDK(t)
		newPathFixture(t, "pnpm")
		s.envContext.UserConfig = &config.Config{Diagnostics: &config.Diagnostics{OnUse: false}}
		out := capturePterm(t)

		s.reportFindings("1.0.0", true)
		if out.Len() != 0 {
			t.Errorf("output = %q, want nothing with diagnostics.onUse off", out.String())
		}
	})
}

// `use` ends with the checks for the version it just selected, and says so
// when another executable is going to run instead.
func TestUseWarnsWhenTheVersionIsShadowed(t *testing.T) {
	s := installedSDK(t)
	f := newPathFixture(t, "pnpm")
	t.Setenv(env.HookFlag, "bash") // a shell with the hook, so that nothing is handed over to a new one
	out := capturePterm(t)

	if err := s.UseWithConfig("1.0.0", env.Session, false); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Now using") || !strings.Contains(got, "comes before vfox in PATH") || !strings.Contains(got, f.shadow) {
		t.Errorf("output = %q, want the selection and the warning about the executable that wins", got)
	}
}
