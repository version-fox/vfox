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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/util"
)

// Diagnose runs the checks of `vfox doctor`. name and version narrow them down
// to one SDK, or to one version of it; both empty checks everything. It only
// reads, so it is safe to run at any time.
func (m *Manager) Diagnose(name string, version sdk.Version) ([]sdk.Finding, error) {
	var findings []sdk.Finding
	findings = append(findings, m.activationFindings()...)

	var sdks []sdk.Sdk
	if name != "" {
		s, err := m.LookupSdk(name)
		if err != nil {
			return nil, fmt.Errorf("%s not supported, error: %w", name, err)
		}
		sdks = append(sdks, s)
	} else {
		all, err := m.LoadAllSdk()
		if err != nil {
			return nil, err
		}
		sdks = all
		findings = append(findings, m.orphanedInstallFindings()...)
	}

	for _, s := range sdks {
		meta := s.Metadata()
		findings = append(findings, pluginFindings(meta)...)

		opts := sdk.DiagnoseOptions{Version: version}
		// Only the version in effect can be shadowed in a way that matters.
		if version != "" {
			opts.PathFor = version
		} else if current := s.Current(); current != "" {
			opts.PathFor = current
		} else {
			opts.NoPath = true
		}
		findings = append(findings, s.Diagnose(opts)...)
	}

	sdk.ApplyIgnore(findings, m.ignores)
	sdk.SortFindings(findings)
	return findings, nil
}

func (m *Manager) ignores(id, sdkName, version string) bool {
	if m.RuntimeEnvContext == nil || m.RuntimeEnvContext.UserConfig == nil {
		return false
	}
	return m.RuntimeEnvContext.UserConfig.Diagnostics.Ignores(id, sdkName, version)
}

// activationFindings checks that this shell went through `vfox activate`.
func (m *Manager) activationFindings() []sdk.Finding {
	if env.IsHookEnv() {
		return nil
	}
	severity := sdk.SeverityError
	if runtime.GOOS == "windows" {
		severity = sdk.SeverityWarn // without a hook only the global scope works there
	}
	return []sdk.Finding{{
		ID:       sdk.FindingPathNotActivated,
		Severity: severity,
		Message:  "vfox is not activated in this shell, so version changes do not reach it.",
		Hints: []string{
			"Add the activation line for your shell to its startup file, for example: eval \"$(vfox activate bash)\"",
			"See https://vfox.dev/guides/quick-start.html",
		},
	}}
}

func pluginFindings(meta *sdk.Metadata) []sdk.Finding {
	if meta == nil || meta.PluginMetadata == nil {
		return nil
	}
	required := meta.PluginMetadata.MinRuntimeVersion
	if required == "" || util.CompareVersion(required, RuntimeVersion) <= 0 {
		return nil
	}
	return []sdk.Finding{{
		ID:       sdk.FindingPluginIncompatible,
		Severity: sdk.SeverityWarn,
		SDK:      strings.ToLower(meta.Name),
		Message:  fmt.Sprintf("The %s plugin needs vfox %s or newer; this is %s.", meta.Name, required, RuntimeVersion),
		Fix:      []string{"vfox upgrade"},
	}}
}

// orphanedInstallFindings reports SDK directories that have installed versions
// but no plugin to manage them.
func (m *Manager) orphanedInstallFindings() []sdk.Finding {
	installs := m.RuntimeEnvContext.PathMeta.Shared.Installs
	plugins := m.RuntimeEnvContext.PathMeta.Shared.Plugins
	entries, err := os.ReadDir(installs)
	if err != nil {
		return nil
	}
	var findings []sdk.Finding
	for _, entry := range entries {
		if !entry.IsDir() || util.FileExists(filepath.Join(plugins, entry.Name())) {
			continue
		}
		children, err := os.ReadDir(filepath.Join(installs, entry.Name()))
		if err != nil {
			continue
		}
		for _, child := range children {
			if child.IsDir() && strings.HasPrefix(child.Name(), "v-") {
				findings = append(findings, sdk.Finding{
					ID:       sdk.FindingPluginMissing,
					Severity: sdk.SeverityWarn,
					SDK:      strings.ToLower(entry.Name()),
					Message:  fmt.Sprintf("%s has installed versions, but its plugin is not added.", entry.Name()),
					Fix:      []string{"vfox add " + entry.Name()},
				})
				break
			}
		}
	}
	return findings
}
