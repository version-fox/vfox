//go:build windows

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
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"

	"github.com/version-fox/vfox/internal/plugin"
)

func TestInstallKeepsMarkerWhenWindowsPayloadIsLocked(t *testing.T) {
	handle := windows.InvalidHandle
	defer func() {
		if handle != windows.InvalidHandle {
			_ = windows.CloseHandle(handle)
		}
	}()
	hookErr := errors.New("installer child failed with an open payload")
	s := newInstallTestSDK(t.TempDir(), func(ctx *plugin.PostInstallHookCtx) error {
		path := filepath.Join(ctx.SdkInfo["test-sdk"].Path, "locked.exe")
		if err := os.WriteFile(path, nil, 0600); err != nil {
			return err
		}
		name, err := windows.UTF16PtrFromString(path)
		if err != nil {
			return err
		}
		handle, err = windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
		if err != nil {
			return err
		}
		return hookErr
	})
	if err := s.Install("1.0"); !errors.Is(err, hookErr) {
		t.Fatalf("Install error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.InstallPath, ".v-1.0.installing")); err != nil {
		t.Fatalf("failed cleanup lost its marker: %v", err)
	}
	if s.CheckRuntimeExist("1.0") || len(s.InstalledList()) != 0 {
		t.Fatal("locked partial payload is exposed as installed")
	}
	if err := windows.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}
	handle = windows.InvalidHandle
	s.plugin.Plugin.(*installTestPlugin).postInstall = nil
	if err := s.Install("1.0"); err != nil {
		t.Fatal(err)
	}
	if !s.CheckRuntimeExist("1.0") {
		t.Fatal("retry did not complete")
	}
}
