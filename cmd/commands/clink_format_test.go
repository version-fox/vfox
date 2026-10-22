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
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pterm/pterm"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/sdk"
)

// The Clink completion script, internal/shell/clink_vfox.lua, completes plugin
// names and installed SDKs by parsing the output of `vfox available` and
// `vfox ls`. Its tests (internal/shell/clink_test.go) run the script against the
// canned output in internal/shell/testdata/clink. These tests check that the
// canned output is what the commands print, so that a change to either format
// is noticed on both sides.

var updateClinkFixtures = flag.Bool("update-clink-fixtures", false,
	"rewrite the output of vfox available and vfox ls in internal/shell/testdata/clink")

// colorSequence is what clink_vfox.lua strips from the output before parsing it.
var colorSequence = regexp.MustCompile("\x1b\\[\\d+m")

func TestAvailableOutputMatchesClinkFixture(t *testing.T) {
	out := printed(func() {
		printAvailable(internal.RegistryIndex{
			{Name: "nodejs", Homepage: "https://github.com/version-fox/vfox-nodejs"},
			{Name: "java", Homepage: "https://github.com/version-fox/vfox-java"},
			{Name: "golang", Homepage: "https://github.com/version-fox/vfox-golang"},
			{Name: "python", Homepage: "https://github.com/version-fox/vfox-python"},
			{Name: "dotnet-core", Homepage: "https://github.com/version-fox/vfox-dotnet"},
			{Name: "my-plugin", Homepage: "https://example.com/vfox-my-plugin"},
		})
	})

	checkClinkFixture(t, "vfox available", "vfox_available.txt", out)
}

func TestInstalledSdksOutputMatchesClinkFixture(t *testing.T) {
	out := printed(func() {
		printInstalledSdks([]sdk.Sdk{
			installedSdk{name: "nodejs", versions: []sdk.Version{"21.5.1", "20.10.0"}},
			installedSdk{name: "java", versions: []sdk.Version{"21.0.1+12"}},
			installedSdk{name: "python"},
			installedSdk{name: "golang", versions: []sdk.Version{"1.22.0"}},
		}, func(string) map[string]string { return nil })
	})

	checkClinkFixture(t, "vfox ls", "vfox_ls.txt", out)
}

// A version can be a link to an SDK that vfox did not install, under any name,
// and can have aliases: the tree adds both to the name of the version.
func TestInstalledSdksWithLinksAndAliasesOutputMatchesClinkFixture(t *testing.T) {
	aliases := map[string]map[string]string{
		"java":   {"sys": "17-sys"},
		"nodejs": {"lts": "21.5.1", "current-lts": "21.5.1"},
	}
	out := printed(func() {
		printInstalledSdks([]sdk.Sdk{
			installedSdk{
				name:     "java",
				versions: []sdk.Version{"21.0.1+12", "17-sys", "system"},
				linked:   map[sdk.Version]string{"17-sys": "/usr/lib/jvm/x", "system": "/usr/lib/jvm/default"},
			},
			installedSdk{name: "nodejs", versions: []sdk.Version{"21.5.1", "20.10.0"}},
			installedSdk{name: "golang", versions: []sdk.Version{"v1.22.0"}},
		}, func(name string) map[string]string { return aliases[name] })
	})

	checkClinkFixture(t, "vfox ls", "vfox_ls_linked.txt", out)
}

// installedSdk is an installed SDK as the tree of vfox ls shows it. Printing the
// tree uses nothing else of an SDK.
type installedSdk struct {
	sdk.Sdk
	name     string
	versions []sdk.Version
	linked   map[sdk.Version]string // the versions that are links, and where they point to
}

func (s installedSdk) Metadata() *sdk.Metadata { return &sdk.Metadata{Name: s.name} }

func (s installedSdk) InstalledList() []sdk.Version { return s.versions }

func (s installedSdk) Installation(version sdk.Version) (*sdk.Installation, error) {
	if target, ok := s.linked[version]; ok {
		return &sdk.Installation{Origin: sdk.OriginLinked, Target: target}, nil
	}
	return &sdk.Installation{Origin: sdk.OriginInstalled}, nil
}

// printed returns what print writes through pterm.
func printed(print func()) string {
	var out bytes.Buffer
	enabled := pterm.Output
	pterm.EnableOutput()
	pterm.SetDefaultOutput(&out)
	defer func() {
		pterm.SetDefaultOutput(os.Stdout)
		if !enabled {
			pterm.DisableOutput()
		}
	}()
	print()
	return out.String()
}

// checkClinkFixture compares the output of a command, once the colors the
// terminal supports are removed, with the file the tests of the Clink script use.
func checkClinkFixture(t *testing.T, command, name, out string) {
	t.Helper()

	plain := colorSequence.ReplaceAllString(out, "")
	if strings.Contains(plain, "\x1b") {
		t.Errorf("the output of %s has escape sequences other than ESC[<number>m, which "+
			"internal/shell/clink_vfox.lua does not strip: %q", command, out)
	}

	path := filepath.Join("..", "..", "internal", "shell", "testdata", "clink", name)
	if *updateClinkFixtures {
		if err := os.WriteFile(path, []byte(plain), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Git may check the file out with CRLF line endings on Windows.
	if want := strings.ReplaceAll(string(want), "\r\n", "\n"); plain != want {
		t.Errorf("the output of %s changed.\ngot:\n%s\nwant (%s):\n%s\n"+
			"internal/shell/clink_vfox.lua parses this output. If the change is intended, check that the script "+
			"still understands it, regenerate the file with\n"+
			"    go test ./cmd/commands -run %s -update-clink-fixtures\n"+
			"and run go test ./internal/shell.",
			command, plain, path, want, t.Name())
	}
}
