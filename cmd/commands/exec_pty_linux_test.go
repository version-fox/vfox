//go:build linux

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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func openLinuxPTY(t *testing.T) (master, slave *os.File) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("open /dev/ptmx: %v", err)
	}
	if err := unix.IoctlSetPointerInt(int(master.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		_ = master.Close()
		t.Skipf("unlock pty: %v", err)
	}
	ptyNum, err := unix.IoctlGetInt(int(master.Fd()), unix.TIOCGPTN)
	if err != nil {
		_ = master.Close()
		t.Skipf("TIOCGPTN: %v", err)
	}
	slave, err = os.OpenFile(fmt.Sprintf("/dev/pts/%d", ptyNum), os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		_ = master.Close()
		t.Skipf("open pty slave: %v", err)
	}
	return master, slave
}

// Reproduces `x=$(vfox exec alpha@1.0.0 -- cmd)` where stdout is piped and
// stderr is attached to an interactive terminal.
func TestExecAutoInstallWithStderrPTYNeverWritesClipboard(t *testing.T) {
	f := newIsolatedCLIFixture(t, false)
	f.writePlugin(t, "alpha", execOutputPlugin)
	writeExecTestTool(t, f.system, "alphatool")
	marker := filepath.Join(f.root, "clipboard.called")
	writeStubClipboardTool(t, f.system, marker)

	master, slave := openLinuxPTY(t)
	defer master.Close()

	exe, err := os.Executable()
	must(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := f.command(ctx, exe, "-test.run=^TestCLIProcess$", "--", "exec", "alpha@1.0.0", "--", "alphatool")
	cmd.Env = append(cmd.Env, "VFOX_TEST_CLI=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = slave
	if err := cmd.Start(); err != nil {
		_ = slave.Close()
		t.Fatal(err)
	}
	_ = slave.Close()

	doneDrain := make(chan struct{})
	go func() {
		defer close(doneDrain)
		_, _ = io.Copy(&stderr, master)
	}()

	waitErr := cmd.Wait()
	select {
	case <-doneDrain:
	case <-ctx.Done():
		_ = master.Close()
		<-doneDrain
	}
	_ = master.Close()
	if waitErr != nil {
		t.Fatalf("exec under stderr PTY failed: %v\nstdout:\n%s\nstderr:\n%s", waitErr, stdout.String(), stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != "cmd-stdout" {
		t.Errorf("exec stdout = %q, want %q", stdout.String(), "cmd-stdout")
	}
	mustNotContain(t, "exec stderr under PTY", stderr.String(), "Copied to clipboard")
	if exists(marker) {
		t.Fatalf("exec with stderr on a PTY invoked clipboard helper (created %q)", marker)
	}
}
