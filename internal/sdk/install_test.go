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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/plugin"
)

type installTestPlugin struct {
	plugin.Plugin
	postInstall func(*plugin.PostInstallHookCtx) error
}

func (p *installTestPlugin) PreInstall(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
	return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: ctx.Version}}, nil
}

func (p *installTestPlugin) HasFunction(name string) bool { return name == "PostInstall" }

func (p *installTestPlugin) PostInstall(ctx *plugin.PostInstallHookCtx) error {
	if p.postInstall != nil {
		return p.postInstall(ctx)
	}
	return nil
}

func newInstallTestSDK(root string, post func(*plugin.PostInstallHookCtx) error) *impl {
	return &impl{
		Name: "test-sdk", InstallPath: root,
		plugin: &plugin.Wrapper{
			Metadata: &plugin.Metadata{Name: "test-sdk"},
			Plugin:   &installTestPlugin{postInstall: post},
		},
	}
}

func TestInstallRejectsIncompleteRuntimes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload bool
		pending bool
		want    bool
	}{
		{name: "empty version directory"},
		{name: "unfinished payload", payload: true, pending: true},
		{name: "legacy installation", payload: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newInstallTestSDK(t.TempDir(), nil)
			path := s.packagePath("1.0")
			if tc.payload {
				path = filepath.Join(path, "test-sdk-1.0")
			}
			if err := os.MkdirAll(path, 0755); err != nil {
				t.Fatal(err)
			}
			if tc.pending {
				if err := os.WriteFile(filepath.Join(s.InstallPath, ".v-1.0.installing"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if got := s.CheckRuntimeExist("1.0"); got != tc.want {
				t.Errorf("CheckRuntimeExist = %v, want %v", got, tc.want)
			}
			if got := len(s.InstalledList()) > 0; got != tc.want {
				t.Errorf("listed installation = %v, want %v", got, tc.want)
			}
			_, err := s.GetRuntimePackage("1.0")
			if (err == nil) != tc.want {
				t.Errorf("GetRuntimePackage error = %v, want installed %v", err, tc.want)
			}
		})
	}
}

func TestInstallBecomesAvailableAfterPostInstall(t *testing.T) {
	var s *impl
	s = newInstallTestSDK(t.TempDir(), func(ctx *plugin.PostInstallHookCtx) error {
		if s.CheckRuntimeExist("1.0") || len(s.InstalledList()) != 0 {
			t.Error("runtime is exposed before PostInstall completes")
		}
		if _, err := s.GetRuntimePackage("1.0"); !errors.Is(err, ErrRuntimeNotFound) {
			t.Errorf("unfinished runtime lookup error = %v", err)
		}
		want := filepath.Join(s.packagePath("1.0"), "test-sdk-1.0")
		if ctx.SdkInfo["test-sdk"].Path != want {
			t.Fatal("PostInstall must receive the final path for embedded paths/shebangs")
		}
		return os.WriteFile(filepath.Join(want, "installed"), []byte(want), 0600)
	})
	if err := s.Install("1.0"); err != nil {
		t.Fatal(err)
	}
	if !s.CheckRuntimeExist("1.0") || len(s.InstalledList()) != 1 {
		t.Fatal("completed installation is not available")
	}
}

func TestInstallFailureCanRetry(t *testing.T) {
	wantErr := errors.New("post-install failed")
	s := newInstallTestSDK(t.TempDir(), func(ctx *plugin.PostInstallHookCtx) error { return wantErr })
	if err := s.Install("1.0"); !errors.Is(err, wantErr) {
		t.Fatalf("Install error = %v, want wrapped hook error", err)
	}
	if _, err := os.Stat(s.packagePath("1.0")); !os.IsNotExist(err) {
		t.Fatalf("failed installation was not removed: %v", err)
	}
	s.plugin.Plugin.(*installTestPlugin).postInstall = nil
	if err := s.Install("1.0"); err != nil {
		t.Fatal(err)
	}
	if !s.CheckRuntimeExist("1.0") {
		t.Fatal("retry did not finish installation")
	}
}

func TestInstallSerializesSameVersion(t *testing.T) {
	root := t.TempDir()
	started, finish := make(chan struct{}), make(chan struct{})
	first := newInstallTestSDK(root, func(ctx *plugin.PostInstallHookCtx) error {
		close(started)
		<-finish
		return nil
	})
	done := make(chan error, 1)
	go func() { done <- first.Install("1.0") }()
	<-started
	second := newInstallTestSDK(root, nil)
	err := second.Install("1.0")
	close(finish)
	if firstErr := <-done; firstErr != nil {
		t.Fatal(firstErr)
	}
	if err == nil {
		t.Fatal("second install must report the active installation, not claim success")
	}
	if err := second.Install("1.0"); err != nil {
		t.Fatalf("completed installation should be reusable: %v", err)
	}
}

// The helper is a separate process so abrupt termination skips all install defers.
func TestInstallInterruptedHelper(t *testing.T) {
	root := os.Getenv("VFOX_INSTALL_INTERRUPTION_TEST")
	if root == "" {
		return
	}
	s := newInstallTestSDK(root, func(ctx *plugin.PostInstallHookCtx) error {
		if err := os.WriteFile(filepath.Join(ctx.SdkInfo["test-sdk"].Path, "partial"), nil, 0600); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, "ready"), nil, 0600); err != nil {
			return err
		}
		time.Sleep(time.Hour)
		return nil
	})
	if err := s.Install("1.0"); err != nil {
		t.Fatal(err)
	}
}

func TestInstallRecoversAfterProcessTermination(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestInstallInterruptedHelper$")
	cmd.Env = append(os.Environ(), "VFOX_INSTALL_INTERRUPTION_TEST="+root)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(root, "ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatal("child did not reach PostInstall")
		}
		time.Sleep(10 * time.Millisecond)
	}
	active := newInstallTestSDK(root, nil)
	if err := active.Install("1.0"); err == nil {
		t.Error("second process did not reject an active installation")
	}
	if _, err := os.Stat(filepath.Join(active.packagePath("1.0"), "test-sdk-1.0", "partial")); err != nil {
		t.Errorf("second process disturbed the active payload: %v", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("terminated install unexpectedly succeeded")
	}
	called := false
	s := newInstallTestSDK(root, func(ctx *plugin.PostInstallHookCtx) error {
		called = true
		_, err := os.Stat(filepath.Join(ctx.SdkInfo["test-sdk"].Path, "partial"))
		if !os.IsNotExist(err) {
			t.Fatalf("retry kept interrupted payload: %v", err)
		}
		return nil
	})
	if s.CheckRuntimeExist("1.0") {
		t.Error("terminated install is marked installed")
	}
	if err := s.Install("1.0"); err != nil {
		t.Fatal(err)
	}
	if !called || !s.CheckRuntimeExist("1.0") {
		t.Fatal("interrupted install was not retried successfully")
	}
}
