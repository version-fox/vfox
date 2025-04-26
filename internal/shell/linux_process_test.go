//go:build linux

package shell

import (
	"os"
	"testing"
)

func TestGetProcess_Linux(t *testing.T) {
	p := GetProcess()
	if p == nil {
		t.Error("GetProcess() returned nil")
	}

	_, ok := p.(linuxProcess)
	if !ok {
		t.Error("GetProcess() returned incorrect type, expected linuxProcess")
	}
}

func TestLinuxProcess_Open(t *testing.T) {
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
			l := linuxProcess{}
			err := l.Open(tt.pid)
			if (err != nil) != tt.wantErr {
				t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLinuxProcess_CommandParsing(t *testing.T) {
	l := linuxProcess{}
	pid := os.Getpid()

	// Test process command retrieval
	err := l.Open(pid)
	if err != nil {
		t.Errorf("Open() failed for current process: %v", err)
	}
}
