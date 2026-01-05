package internal

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

func detectPidNotExists(t *testing.T, pid int32, num int32) int32 {
	if num < 1 {
		t.Fatalf("num must be greater than 1")
	}

	for i := pid + 1; i < pid+num; i++ {
		exists, err := process.PidExists(i)
		if err != nil {
			t.Fatalf("failed to check pid %d existence: %v", i, err)
		}

		if !exists {
			return i
		}
	}

	return pid + num
}

func TestCleanTmp(t *testing.T) {
	tmpRoot := filepath.Join(os.TempDir(), "vfox-cleantmp-test")
	_ = os.RemoveAll(tmpRoot)
	defer os.RemoveAll(tmpRoot)
	now := time.Now()
	today := strconv.FormatInt(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix(), 10)
	// Create a simulated temporary directory structure
	yesterday := strconv.FormatInt(now.Add(-250*time.Hour).Unix(), 10)
	pid := os.Getpid()
	otherPid := int(detectPidNotExists(t, int32(pid+100), 10000))
	dirs := []string{
		filepath.Join(tmpRoot, today+"-"+strconv.Itoa(pid)),
		// filepath.Join(tmpRoot, yesterday+"-"+strconv.Itoa(pid)),
		filepath.Join(tmpRoot, yesterday+"-"+strconv.Itoa(otherPid)),
		filepath.Join(tmpRoot, yesterday+"-"+strconv.Itoa(pid)),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}
	cleanFlagPath := filepath.Join(tmpRoot, cleanupFlagFilename)
	// Write yesterday to cleanFlagPath
	if err := os.WriteFile(cleanFlagPath, []byte(yesterday), 0o644); err != nil {
		t.Fatalf("failed to write cleanFlagPath: %v", err)
	}

	// Construct Manager
	m := &Manager{PathMeta: &PathMeta{
		User:    UserPaths{Temp: tmpRoot},
		Working: WorkingPaths{SessionShim: tmpRoot},
	}}

	// Execute cleanup
	m.CleanTmp()

	// today-pid should be retained
	if _, err := os.Stat(filepath.Join(tmpRoot, today+"-"+strconv.Itoa(pid))); os.IsNotExist(err) {
		t.Errorf("today-pid dir should exist, but got removed")
	}

	// yesterday-otherPid should be deleted
	if _, err := os.Stat(filepath.Join(tmpRoot, yesterday+"-"+strconv.Itoa(otherPid))); err == nil {
		t.Errorf("yesterday-otherPid dir should be removed, but still exists")
	}

	// yesterday-pid should be retained
	if _, err := os.Stat(filepath.Join(tmpRoot, yesterday+"-"+strconv.Itoa(pid))); os.IsNotExist(err) {
		t.Errorf("yesterday-pid dir should exist, but got removed")
	}
}
