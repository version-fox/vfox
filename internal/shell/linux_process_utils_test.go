//go:build linux

package shell

import (
	"os"
	"strings"
	"testing"
)

func TestGetProcessPath_Linux(t *testing.T) {
	p := GetProcessUtils()
	if p == nil {
		t.Error("GetProcessUtils() returned nil")
	}

	_, ok := p.(linuxProcessUtils)
	if !ok {
		t.Error("GetProcessUtils() returned incorrect type, expected linuxProcessUtils")
	}
}

func TestLinuxProcessUtils_GetPath(t *testing.T) {
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

	l := linuxProcessUtils{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := l.GetPath(tt.pid)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !strings.Contains(path, "/") {
				t.Errorf("GetPath() returned invalid path: %v", path)
			}
		})
	}
}
