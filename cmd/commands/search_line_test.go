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
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/printer"
	"github.com/version-fox/vfox/internal/shared/util"
)

// recordingInstaller records the versions it is asked to install.
type recordingInstaller struct {
	installed []sdk.Version
}

func (r *recordingInstaller) Install(version sdk.Version) error {
	r.installed = append(r.installed, version)
	return nil
}

// failingReader fails every read, as a terminal that went away does.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("terminal gone")
}

func searchOptions() []*printer.KV {
	return []*printer.KV{
		{Key: "18.20.8", Value: "18.20.8 (LTS Hydrogen)"},
		{Key: "20.19.0", Value: "20.19.0 (LTS Iron)"},
	}
}

func TestInstallByLineInstallsTheChosenVersion(t *testing.T) {
	var installer recordingInstaller
	var out bytes.Buffer
	err := installByLine(&installer, strings.NewReader("2\n"), &out, searchOptions(), util.NewSet[string](), "nodejs")
	if err != nil {
		t.Fatalf("installByLine() error: %v", err)
	}
	if len(installer.installed) != 1 || installer.installed[0] != "20.19.0" {
		t.Errorf("installed %v, want [20.19.0]", installer.installed)
	}
	if !strings.Contains(out.String(), "Please select a version of nodejs to install") {
		t.Errorf("prompt does not name the SDK:\n%s", out.String())
	}
}

func TestInstallByLineCancelInstallsNothing(t *testing.T) {
	var installer recordingInstaller
	var out bytes.Buffer
	if err := installByLine(&installer, strings.NewReader("\n"), &out, searchOptions(), util.NewSet[string](), "nodejs"); err != nil {
		t.Fatalf("installByLine() error: %v", err)
	}
	if len(installer.installed) != 0 {
		t.Errorf("installed %v after a cancelled selection, want nothing", installer.installed)
	}
}

func TestInstallByLineRefusesAnInstalledVersion(t *testing.T) {
	var installer recordingInstaller
	var out bytes.Buffer
	installed := util.NewSet[string]()
	installed.Add("20.19.0")
	err := installByLine(&installer, strings.NewReader("2\n1\n"), &out, searchOptions(), installed, "nodejs")
	if err != nil {
		t.Fatalf("installByLine() error: %v", err)
	}
	if len(installer.installed) != 1 || installer.installed[0] != "18.20.8" {
		t.Errorf("installed %v, want [18.20.8] after the installed version was refused", installer.installed)
	}
	if !strings.Contains(out.String(), "20.19.0 is already installed") {
		t.Errorf("output does not refuse the installed version:\n%s", out.String())
	}
}

func TestInstallByLineReportsReadError(t *testing.T) {
	var installer recordingInstaller
	var out bytes.Buffer
	err := installByLine(&installer, failingReader{}, &out, searchOptions(), util.NewSet[string](), "nodejs")
	if err == nil || !strings.Contains(err.Error(), "select version error") {
		t.Fatalf("installByLine() error = %v, want a select version error", err)
	}
	if len(installer.installed) != 0 {
		t.Errorf("installed %v after a failed read, want nothing", installer.installed)
	}
}
