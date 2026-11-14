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
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal/shared/printer"
	"github.com/version-fox/vfox/internal/shared/util"
	"golang.org/x/sys/windows"
)

// TestGitBashMinttySelection runs the terminal code in a Git Bash window: the
// test binary runs under Git for Windows' bash.exe in a mintty window, as it
// does for a person at mintty. scripts/gitbash-mintty-test.ps1 starts it and
// types the answers. When the test waits for one it creates a marker file in
// VFOX_TEST_GITBASH_MINTTY_DIR, and it writes its result there too.
//
// VFOX_TEST_GITBASH_MINTTY_STDIO says what the window gives the program: "pipe"
// for a pseudo terminal that reaches it as a pipe (the line input of the fix),
// or "console" for a pseudo console (the arrow-key selection).
func TestGitBashMinttySelection(t *testing.T) {
	dir := os.Getenv("VFOX_TEST_GITBASH_MINTTY_DIR")
	if dir == "" {
		t.Skip("only runs in a Git Bash mintty window, started by scripts/gitbash-mintty-test.ps1")
	}
	stdio := os.Getenv("VFOX_TEST_GITBASH_MINTTY_STDIO")
	if stdio != "pipe" && stdio != "console" {
		t.Fatalf("VFOX_TEST_GITBASH_MINTTY_STDIO = %q, want pipe or console", stdio)
	}
	var report strings.Builder
	fmt.Fprintf(&report, "MSYS: %q\n", os.Getenv("MSYS"))
	fmt.Fprintf(&report, "stdin: %s\n", gitBashHandleKind(os.Stdin.Fd()))
	fmt.Fprintf(&report, "stdout: %s\n", gitBashHandleKind(os.Stdout.Fd()))
	isTTY, canReadKeys := util.IsTTY(), util.CanReadKeys()
	fmt.Fprintf(&report, "IsTTY: %v\nCanReadKeys: %v\n", isTTY, canReadKeys)

	markWaiting(t, dir, "select")
	version, selectErr := selectFromOptions([]string{"18.20.8", "20.19.0"}, "nodejs")
	fmt.Fprintf(&report, "selected: %q, error: %v\n", version, selectErr)

	markWaiting(t, dir, "confirm")
	confirmed, confirmErr := printer.Confirm(&pterm.DefaultInteractiveConfirm, "Proceed with the check?")
	fmt.Fprintf(&report, "confirmed: %v, error: %v\n", confirmed, confirmErr)

	ok := isTTY && selectErr == nil && version == "20.19.0" && confirmErr == nil && confirmed
	if stdio == "pipe" {
		ok = ok && !canReadKeys
	} else {
		ok = ok && canReadKeys
	}
	if err := os.WriteFile(filepath.Join(dir, "result.txt"), []byte(report.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Errorf("Git Bash in mintty, %s:\n%s", stdio, report.String())
	}
}

// markWaiting tells the driver that the test is about to wait for the input of step.
func markWaiting(t *testing.T, dir, step string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, step+".ready"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

// gitBashHandleKind describes a standard handle: the name of a pipe, or the file type.
func gitBashHandleKind(fd uintptr) string {
	fileType, err := windows.GetFileType(windows.Handle(fd))
	if err != nil {
		return "error: " + err.Error()
	}
	if fileType != windows.FILE_TYPE_PIPE {
		return fmt.Sprintf("file type %d", fileType)
	}
	return "pipe " + gitBashPipeName(fd)
}

// gitBashPipeName returns the name of a pipe as FileNameInfo reports it.
func gitBashPipeName(fd uintptr) string {
	buf := make([]byte, 4+2*260)
	if err := windows.GetFileInformationByHandleEx(windows.Handle(fd), windows.FileNameInfo, &buf[0], uint32(len(buf))); err != nil {
		return "error: " + err.Error()
	}
	size := int(binary.LittleEndian.Uint32(buf[:4]))
	if size > len(buf)-4 {
		size = len(buf) - 4
	}
	units := make([]uint16, size/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(buf[4+2*i:])
	}
	return string(utf16.Decode(units))
}
