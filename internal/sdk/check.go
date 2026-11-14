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
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/version-fox/vfox/internal/env"
)

// DiagnoseOptions selects what Diagnose looks at.
type DiagnoseOptions struct {
	// Version limits the checks to one version. Empty means every version on
	// disk, including interrupted installations and broken links.
	Version Version
	// Fast keeps only the checks that are cheap enough to run on every
	// `vfox use`, and stays quiet about things `use` cannot be affected by.
	Fast bool
	// NoPath skips the checks that depend on the PATH of the calling shell.
	NoPath bool
	// PathFor limits the PATH checks to one version, the one in effect: what
	// sits in front of the other installed versions does not matter. Empty
	// means every version that is looked at.
	PathFor Version
}

// Diagnose reports what looks wrong with the installed versions. It only
// reads: nothing it does changes an installation.
func (b *impl) Diagnose(opts DiagnoseOptions) []Finding {
	versions := []Version{opts.Version}
	if opts.Version == "" {
		versions = b.versionsOnDisk()
	}
	var findings []Finding
	for _, version := range versions {
		findings = append(findings, b.diagnoseVersion(version, opts)...)
	}
	ApplyIgnore(findings, b.ignores)
	SortFindings(findings)
	return findings
}

// versionsOnDisk lists every version that has a directory or an installation
// marker, whether or not it is complete.
func (b *impl) versionsOnDisk() []Version {
	entries, err := os.ReadDir(b.InstallPath)
	if err != nil {
		return nil
	}
	seen := make(map[Version]struct{})
	for _, entry := range entries {
		name := entry.Name()
		switch {
		case entry.IsDir() && strings.HasPrefix(name, packageInstalledPrefix):
			seen[Version(strings.TrimPrefix(name, packageInstalledPrefix))] = struct{}{}
		case strings.HasPrefix(name, ".v-") && strings.HasSuffix(name, ".installing"):
			seen[Version(strings.TrimSuffix(strings.TrimPrefix(name, ".v-"), ".installing"))] = struct{}{}
		}
	}
	versions := make([]Version, 0, len(seen))
	for version := range seen {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })
	return versions
}

func (b *impl) newFinding(version Version, id string, severity Severity, message string) Finding {
	return Finding{
		ID:       id,
		Severity: severity,
		SDK:      strings.ToLower(b.Name),
		Version:  string(version),
		Message:  message,
	}
}

func (b *impl) diagnoseVersion(version Version, opts DiagnoseOptions) []Finding {
	label := b.Label(version)

	if b.hasPendingInstall(version) {
		if opts.Fast {
			return nil // `use` cannot select it
		}
		f := b.newFinding(version, FindingInstallInterrupted, SeverityWarn, label+" was not installed completely: an installation was interrupted.")
		f.Fix = []string{"vfox install " + label}
		return []Finding{f}
	}

	if !b.CheckRuntimeExist(version) {
		if b.brokenLink(version) {
			target, _ := b.linkedTarget(version)
			f := b.newFinding(version, FindingLinkDangling, SeverityError,
				fmt.Sprintf("%s is linked to %s, which no longer exists.", label, target))
			f.Detail = map[string]string{"target": target}
			f.Fix = []string{"vfox uninstall " + label}
			return []Finding{f}
		}
		if _, err := os.Stat(b.packagePath(version)); err == nil {
			f := b.newFinding(version, FindingPayloadEmpty, SeverityError,
				fmt.Sprintf("%s has a directory but no runtime in it: %s.", label, b.packagePath(version)))
			f.Hints = []string{"Remove that directory by hand, then install the version again."}
			return []Finding{f}
		}
		return nil
	}

	inst, err := b.Installation(version)
	if err != nil {
		return []Finding{b.newFinding(version, FindingPayloadEmpty, SeverityError,
			fmt.Sprintf("%s cannot be read: %v.", label, err))}
	}

	var findings []Finding
	if !opts.Fast && inst.Receipt == nil {
		f := b.newFinding(version, FindingNoReceipt, SeverityInfo,
			label+" was installed before vfox kept receipts, so changes made to it cannot be detected.")
		f.Fix = []string{"vfox doctor --adopt " + label}
		findings = append(findings, f)
	}

	payload, dirs := b.payloadFindings(version, inst)
	findings = append(findings, payload...)
	findings = append(findings, b.driftFindings(version, inst, opts)...)
	if !opts.NoPath && (opts.PathFor == "" || opts.PathFor == version) && len(dirs) > 0 {
		findings = append(findings, b.shadowFindings(version, dirs)...)
	}
	return findings
}

// payloadFindings checks that the directories the plugin puts on PATH exist and
// hold commands. It returns those directories for the checks that follow.
func (b *impl) payloadFindings(version Version, inst *Installation) ([]Finding, []string) {
	label := b.Label(version)
	envs, err := b.EnvKeys(inst.RuntimePackage)
	if err != nil {
		return []Finding{b.newFinding(version, FindingEnvKeysFailed, SeverityError,
			fmt.Sprintf("%s: the plugin could not say what to put in the environment: %v.", label, err))}, nil
	}
	dirs := envs.Paths.Slice()
	if len(dirs) == 0 {
		return nil, nil // the plugin only sets variables
	}
	if len(env.ListExecutables(dirs)) > 0 {
		return nil, dirs
	}
	severity := SeverityError
	for _, dir := range dirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			severity = SeverityWarn // the directory is there, the commands are not
			break
		}
	}
	f := b.newFinding(version, FindingPayloadEmpty, severity,
		fmt.Sprintf("%s has no command in %s.", label, strings.Join(dirs, ", ")))
	f.Detail = map[string]string{"directories": strings.Join(dirs, ", ")}
	if inst.Origin == OriginLinked {
		f.Hints = []string{"Check that the linked directory is a complete installation: " + inst.Target}
	} else {
		f.Fix = []string{"vfox install --reinstall " + label}
	}
	return []Finding{f}, nil
}

// factDiff is one fact that is no longer what vfox recorded.
type factDiff struct {
	Key, Was, Now string
	Missing       bool // the plugin no longer reports it
}

func compareFacts(recorded, current map[string]string) []factDiff {
	var diffs []factDiff
	for key, was := range recorded {
		now, ok := current[key]
		if !ok {
			diffs = append(diffs, factDiff{Key: key, Was: was, Missing: true})
		} else if now != was {
			diffs = append(diffs, factDiff{Key: key, Was: was, Now: now})
		}
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Key < diffs[j].Key })
	return diffs
}

func shortFact(value string) string {
	const max = 12
	if len(value) > max {
		return value[:max] + "…"
	}
	return value
}

// driftFindings compares the facts recorded at installation with the ones the
// plugin reports now.
func (b *impl) driftFindings(version Version, inst *Installation, opts DiagnoseOptions) []Finding {
	if inst.Receipt == nil || len(inst.Receipt.Facts) == 0 || b.plugin == nil || !b.plugin.HasFunction("Fingerprint") {
		return nil
	}
	label := b.Label(version)
	current, err := b.fingerprint(version, sdkInfoOf(inst.RuntimePackage), inst.Runtime.Path)
	if err != nil {
		if opts.Fast {
			return nil
		}
		return []Finding{b.newFinding(version, FindingFingerprintFailed, SeverityWarn,
			fmt.Sprintf("%s could not be compared with what vfox recorded: %v.", label, err))}
	}
	diffs := compareFacts(inst.Receipt.Facts, current)
	if len(diffs) == 0 {
		return nil
	}

	parts := make([]string, 0, len(diffs))
	detail := make(map[string]string, len(diffs))
	for _, diff := range diffs {
		now := shortFact(diff.Now)
		if diff.Missing {
			now = "unreadable"
		}
		parts = append(parts, fmt.Sprintf("%s: %s -> %s", diff.Key, shortFact(diff.Was), now))
		detail[diff.Key] = diff.Was + " -> " + diff.Now
	}
	summary := strings.Join(parts, "; ")

	if inst.Origin == OriginLinked {
		f := b.newFinding(version, FindingDrift, SeverityInfo,
			fmt.Sprintf("%s has changed since it was linked (%s).", label, summary))
		f.Detail = detail
		f.Hints = []string{"vfox does not own a linked SDK; this is the owner's change."}
		return []Finding{f}
	}
	f := b.newFinding(version, FindingDrift, SeverityWarn,
		fmt.Sprintf("%s has changed since vfox installed it (%s).", label, summary))
	f.Detail = detail
	f.Hints = []string{
		"Something modified the SDK in place, for example a self-update command. vfox manages versions by directory and does not track that.",
		"To move to a newer version, install it with vfox instead: vfox install " + strings.ToLower(b.Name) + "@<new-version>",
	}
	f.Fix = []string{"vfox install --reinstall " + label}
	return []Finding{f}
}

// shadowFindings reports the commands of the version that an executable in
// front of vfox's directories keeps from running.
func (b *impl) shadowFindings(version Version, dirs []string) []Finding {
	if b.envContext == nil {
		return nil
	}
	shadows := b.envContext.FindShadows(env.ListExecutables(dirs))
	if len(shadows) == 0 {
		return nil
	}
	label := b.Label(version)

	var order []string
	byDir := make(map[string][]env.Shadow)
	for _, shadow := range shadows {
		if _, seen := byDir[shadow.Dir]; !seen {
			order = append(order, shadow.Dir)
		}
		byDir[shadow.Dir] = append(byDir[shadow.Dir], shadow)
	}

	var findings []Finding
	for _, dir := range order {
		group := byDir[dir]
		names := make([]string, len(group))
		for i, shadow := range group {
			names[i] = shadow.Name
		}
		source := group[0].Source

		severity := SeverityWarn
		if source == env.ShadowSourceActivatedEnv || source == env.ShadowSourcePreserved {
			severity = SeverityInfo // kept in front on purpose
		}
		var message string
		if len(group) == 1 {
			message = fmt.Sprintf("%s runs %s instead of %s: its directory comes before vfox in PATH.",
				commandList(names), group[0].Winner, label)
		} else {
			message = fmt.Sprintf("%s run files from %s instead of %s: that directory comes before vfox in PATH.",
				commandList(names), dir, label)
		}
		f := b.newFinding(version, FindingPathShadowed, severity, message)
		f.Detail = map[string]string{
			"directory": dir,
			"commands":  strings.Join(names, ", "),
			"winner":    group[0].Winner,
			"source":    string(source),
		}
		f.Hints = append(f.Hints, b.shadowHints(group[0].Winner)...)
		f.Hints = append(f.Hints, shadowSourceHint(source)...)
		findings = append(findings, f)
	}
	return findings
}

// shadowHints returns the explanations the plugin declared for the executable.
func (b *impl) shadowHints(winner string) []string {
	if b.plugin == nil || b.plugin.Metadata == nil {
		return nil
	}
	var hints []string
	lower := strings.ToLower(winner)
	for _, hint := range b.plugin.ShadowHints {
		if hint.Match == "" || !strings.Contains(lower, strings.ToLower(hint.Match)) {
			continue
		}
		text := hint.Message
		if hint.URL != "" {
			text += " See " + hint.URL
		}
		if text != "" {
			hints = append(hints, text)
		}
	}
	return hints
}

func shadowSourceHint(source env.ShadowSource) []string {
	switch source {
	case env.ShadowSourceInherited:
		// Only the legacy mode keeps such a directory in front; the default moves it behind vfox's.
		return []string{"That directory was already in PATH when vfox was activated, and `path.mode` is `legacy`, which keeps it in front. `vfox config path.mode baseline` puts such directories behind vfox's."}
	case env.ShadowSourceAdded:
		return []string{"That directory was added to PATH after `vfox activate` ran. Activate vfox after the line that adds it in your shell's startup file, or run `vfox config path.mode aggressive`."}
	case env.ShadowSourceUnknown:
		return []string{"vfox did not record the PATH it found when it was activated. Open a new shell, or run `vfox config path.mode aggressive`."}
	default:
		return nil
	}
}
