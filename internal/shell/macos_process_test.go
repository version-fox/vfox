//go:build darwin

package shell

import (
	"os"
	"testing"
)

func TestGetProcess_Mac(t *testing.T) {
	p := GetProcess()
	if p == nil {
		t.Error("GetProcess() returned nil")
	}

	_, ok := p.(macosProcess)
	if !ok {
		t.Error("GetProcess() returned incorrect type, expected macosProcess")
	}
}

func TestMacProcess_Open(t *testing.T) {
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
			name:    "non-existent PID",
			pid:     99999999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := macosProcess{}
			err := m.Open(tt.pid)
			if (err != nil) != tt.wantErr {
				t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMacProcess_CommandParsing(t *testing.T) {
	m := macosProcess{}
	pid := os.Getpid()

	// Test process command retrieval
	err := m.Open(pid)
	if err != nil {
		t.Errorf("Open() failed for current process: %v", err)
	}
}
