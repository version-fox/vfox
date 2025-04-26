//go:build windows

package shell

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func TestGetProcess(t *testing.T) {
	p := GetProcess()
	if p == nil {
		t.Error("GetProcess() returned nil")
	}

	_, ok := p.(windowsProcess)
	if !ok {
		t.Error("GetProcess() returned incorrect type, expected windowsProcess")
	}
}

func TestWindowsProcess_Open(t *testing.T) {
	tests := []struct {
		name    string
		pid     int
		wantErr bool
	}{
		{
			name:    "invalid PID",
			pid:     -1,
			wantErr: true,
		},
		{
			name:    "current process PID",
			pid:     os.Getpid(),
			wantErr: false,
		},
		{
			name:    "system idle process",
			pid:     0,
			wantErr: true, // Should fail due to insufficient privileges
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := windowsProcess{}
			err := w.Open(tt.pid)
			if (err != nil) != tt.wantErr {
				t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWindowsProcess_OpenProcessHandling(t *testing.T) {
	w := windowsProcess{}

	// Test process handle cleanup
	pid := os.Getpid()
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		t.Fatalf("Failed to open process: %v", err)
	}

	// Verify handle is closed after Open() returns
	err = w.Open(pid)
	if err != nil {
		t.Errorf("Open() failed: %v", err)
	}

	// Try to close the handle again - should fail if already closed
	err = windows.CloseHandle(hProcess)
	if err == nil {
		t.Error("Handle was not properly closed")
	}
}

func TestWindowsProcess_PathRetrieval(t *testing.T) {
	pid := os.Getpid()

	// Open process and get path
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		t.Fatalf("Failed to open process: %v", err)
	}
	defer windows.CloseHandle(hProcess)

	var exePath [windows.MAX_PATH]uint16
	size := uint32(len(exePath))
	err = windows.QueryFullProcessImageName(hProcess, 0, &exePath[0], &size)
	if err != nil {
		t.Errorf("Failed to query process path: %v", err)
	}

	path := windows.UTF16ToString(exePath[:size])
	if path == "" {
		t.Error("Retrieved path is empty")
	}
}
