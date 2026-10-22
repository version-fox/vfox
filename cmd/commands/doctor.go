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

package commands

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/sdk"
)

var Doctor = &cli.Command{
	Name:      "doctor",
	Usage:     "Check vfox and the installed SDKs for problems",
	ArgsUsage: "[<sdk>[@<version>]]",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "json",
			Usage: "Output the findings as JSON",
		},
		&cli.BoolFlag{
			Name:  "adopt",
			Usage: "Record the current state of versions installed before receipts existed, as the state to compare against later",
		},
	},
	Action:   doctorCmd,
	Category: CategorySDK,
}

func doctorCmd(ctx context.Context, cmd *cli.Command) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	name, version := splitDoctorSpec(cmd.Args().First())
	if name != "" && version != "" {
		if source, err := manager.LookupSdk(name); err == nil {
			version = manager.ResolveVersionName(name, source, version)
		}
	}

	if cmd.Bool("adopt") {
		return doctorAdopt(cmd, manager, name, version)
	}

	findings, err := manager.Diagnose(name, version, versionInEffect(manager))
	if err != nil {
		return err
	}
	if cmd.Bool("json") {
		if err := writeJSON(cmd, struct {
			OK       bool          `json:"ok"`
			Findings []sdk.Finding `json:"findings"`
		}{OK: !hasErrors(findings), Findings: nonNil(findings)}); err != nil {
			return err
		}
	} else if err := renderFindings(cmd.Writer, findings); err != nil {
		return err
	}
	if hasErrors(findings) {
		return cli.Exit("", 1)
	}
	return nil
}

// splitDoctorSpec reads [<sdk>[@<version>]]. The version is returned as typed;
// see Manager.ResolveVersionName for what happens to a leading 'v'.
func splitDoctorSpec(spec string) (name string, version sdk.Version) {
	name, v, _ := strings.Cut(spec, "@")
	return strings.ToLower(name), sdk.Version(v)
}

func nonNil(findings []sdk.Finding) []sdk.Finding {
	if findings == nil {
		return []sdk.Finding{}
	}
	return findings
}

// hasErrors reports whether a finding the user has not silenced is an error.
func hasErrors(findings []sdk.Finding) bool {
	for _, f := range findings {
		if !f.Ignored && f.Severity == sdk.SeverityError {
			return true
		}
	}
	return false
}

// renderFindings writes the findings for a person to read.
func renderFindings(w io.Writer, findings []sdk.Finding) error {
	var errs, warns, infos, ignored int
	for _, f := range findings {
		switch {
		case f.Ignored:
			ignored++
		case f.Severity == sdk.SeverityError:
			errs++
		case f.Severity == sdk.SeverityWarn:
			warns++
		default:
			infos++
		}
	}
	if errs+warns+infos == 0 {
		if _, err := fmt.Fprintln(w, "No problems found."); err != nil {
			return err
		}
	} else {
		summary := fmt.Sprintf("Found %s, %s and %s.", plural(errs, "error"), plural(warns, "warning"), plural(infos, "notice"))
		if _, err := fmt.Fprintln(w, summary); err != nil {
			return err
		}
	}
	if ignored > 0 {
		if _, err := fmt.Fprintf(w, "%s ignored by diagnostics.ignore.\n", plural(ignored, "finding")); err != nil {
			return err
		}
	}
	for _, f := range findings {
		subject := f.SDK
		if f.Version != "" {
			subject += "@" + f.Version
		}
		header := fmt.Sprintf("\n%-5s  %s", f.Severity, f.ID)
		if subject != "" {
			header += "  " + subject
		}
		if f.Ignored {
			header += "  (ignored)"
		}
		if _, err := fmt.Fprintln(w, header); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "       %s\n", f.Message); err != nil {
			return err
		}
		for _, hint := range f.Hints {
			if _, err := fmt.Fprintf(w, "       %s\n", hint); err != nil {
				return err
			}
		}
		for _, fix := range f.Fix {
			if _, err := fmt.Fprintf(w, "       fix: %s\n", fix); err != nil {
				return err
			}
		}
	}
	return nil
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// doctorAdopt writes receipts for the versions that have none.
func doctorAdopt(cmd *cli.Command, manager *internal.Manager, name string, version sdk.Version) error {
	var sdks []sdk.Sdk
	if name != "" {
		source, err := manager.LookupSdk(name)
		if err != nil {
			return fmt.Errorf("%s not supported, error: %w", name, err)
		}
		sdks = append(sdks, source)
	} else {
		all, err := manager.LoadAllSdk()
		if err != nil {
			return err
		}
		sdks = all
	}

	written, failed := 0, 0
	for _, source := range sdks {
		versions := source.InstalledList()
		if version != "" {
			versions = []sdk.Version{version}
		}
		for _, v := range versions {
			label := fmt.Sprintf("%s@%s", strings.ToLower(source.Metadata().Name), v)
			adopted, err := source.Adopt(v)
			switch {
			case err != nil:
				failed++
				fmt.Fprintf(cmd.ErrWriter, "Could not adopt %s: %v\n", label, err)
			case adopted:
				written++
				fmt.Fprintf(cmd.Writer, "Adopted %s\n", label)
			}
		}
	}
	fmt.Fprintf(cmd.Writer, "%s written.\n", plural(written, "receipt"))
	if failed > 0 {
		return cli.Exit("", 1)
	}
	return nil
}
