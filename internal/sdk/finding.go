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
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
)

// Severity ranks a Finding.
type Severity string

const (
	SeverityInfo  Severity = "info"
	SeverityWarn  Severity = "warn"
	SeverityError Severity = "error"
)

func (s Severity) rank() int {
	switch s {
	case SeverityError:
		return 2
	case SeverityWarn:
		return 1
	default:
		return 0
	}
}

// AtLeast reports whether s is as severe as min or more.
func (s Severity) AtLeast(min Severity) bool {
	return s.rank() >= min.rank()
}

// The IDs of the checks. They are stable: `diagnostics.ignore` and
// `vfox doctor --json` refer to them.
const (
	FindingPathShadowed       = "path.shadowed"
	FindingPathNotActivated   = "path.not-activated"
	FindingDrift              = "sdk.drift"
	FindingFingerprintFailed  = "sdk.fingerprint-failed"
	FindingLinkDangling       = "sdk.link-dangling"
	FindingPayloadEmpty       = "sdk.payload-empty"
	FindingEnvKeysFailed      = "sdk.envkeys-failed"
	FindingNoReceipt          = "sdk.no-receipt"
	FindingInstallInterrupted = "sdk.install-interrupted"
	FindingPluginMissing      = "sdk.plugin-missing"
	FindingPluginIncompatible = "plugin.incompatible"
)

// Finding is one thing a check found.
type Finding struct {
	ID       string            `json:"id"`
	Severity Severity          `json:"severity"`
	SDK      string            `json:"sdk,omitempty"`
	Version  string            `json:"version,omitempty"`
	Message  string            `json:"message"`
	Detail   map[string]string `json:"detail,omitempty"`
	Hints    []string          `json:"hints,omitempty"` // explanations
	Fix      []string          `json:"fix,omitempty"`   // commands that resolve it
	Ignored  bool              `json:"ignored,omitempty"`
}

// ApplyIgnore marks the findings the user chose to silence.
func ApplyIgnore(findings []Finding, ignores func(id, sdk, version string) bool) {
	if ignores == nil {
		return
	}
	for i := range findings {
		f := &findings[i]
		if ignores(f.ID, f.SDK, f.Version) {
			f.Ignored = true
		}
	}
}

// SortFindings orders findings by SDK and version, the most severe first.
func SortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.SDK != b.SDK {
			return a.SDK < b.SDK
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		if a.Severity != b.Severity {
			return a.Severity.rank() > b.Severity.rank()
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.Message < b.Message
	})
}

// diagnosticsOnUse reports whether `vfox use` runs the fast checks by itself.
func (b *impl) diagnosticsOnUse() bool {
	if b.envContext == nil || b.envContext.UserConfig == nil || b.envContext.UserConfig.Diagnostics == nil {
		return true
	}
	return b.envContext.UserConfig.Diagnostics.OnUse
}

func (b *impl) ignores(id, sdk, version string) bool {
	if b.envContext == nil || b.envContext.UserConfig == nil {
		return false
	}
	return b.envContext.UserConfig.Diagnostics.Ignores(id, sdk, version)
}

// reportFindings runs the fast checks for the version that was just selected
// and tells the user what deserves attention. Nothing it finds fails `use`.
func (b *impl) reportFindings(version Version, hookEnv bool) {
	if !b.diagnosticsOnUse() {
		return
	}
	findings := b.Diagnose(DiagnoseOptions{Version: version, Fast: true, NoPath: !hookEnv})
	PrintFindings(findings, SeverityWarn)
}

// PrintFindings prints the findings that are at least as severe as min and not ignored.
func PrintFindings(findings []Finding, min Severity) {
	printFindings(nil, findings, min)
}

// printFindings writes to w, or to pterm's output when w is nil.
func printFindings(w io.Writer, findings []Finding, min Severity) {
	warning, failure := pterm.Warning, pterm.Error
	if w != nil {
		warning, failure = *pterm.Warning.WithWriter(w), *pterm.Error.WithWriter(w)
	}
	text := func(s string) {
		if w != nil {
			pterm.Fprintln(w, s)
			return
		}
		pterm.Println(s)
	}
	for _, f := range findings {
		if f.Ignored || !f.Severity.AtLeast(min) {
			continue
		}
		if f.Severity == SeverityError {
			failure.Println(f.Message)
		} else {
			warning.Println(f.Message)
		}
		for _, hint := range f.Hints {
			text("  " + hint)
		}
		for _, fix := range f.Fix {
			text("  Fix: " + pterm.LightBlue(fix))
		}
	}
}

// commandList renders command names for a sentence: `node`, `npm` and 3 more.
func commandList(names []string) string {
	const shown = 3
	quoted := make([]string, 0, shown)
	for i, name := range names {
		if i == shown {
			break
		}
		quoted = append(quoted, "`"+name+"`")
	}
	switch {
	case len(names) > shown:
		return strings.Join(quoted, ", ") + " and " + strconv.Itoa(len(names)-shown) + " more"
	case len(quoted) > 1:
		return strings.Join(quoted[:len(quoted)-1], ", ") + " and " + quoted[len(quoted)-1]
	default:
		return strings.Join(quoted, "")
	}
}
