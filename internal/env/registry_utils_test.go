package env

import (
	"reflect"
	"testing"
)

func TestDedupOrderedPaths(t *testing.T) {
	inputs := []string{
		"C:/Foo",
		"C:/Foo/../Foo",
		"C:/Bar",
		"c:/bar",
		"  ",
		"",
	}

	got := dedupOrderedPaths(inputs)
	want := []string{"C:/Foo", "C:/Bar"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Dedup mismatch: got %v want %v", got, want)
	}
}

func TestRemovePaths(t *testing.T) {
	existing := []string{"C:/Foo", "C:/Bar", "C:/Baz"}
	toRemove := []string{"c:/bar", "C:/foo", "  "}

	got := removePaths(existing, toRemove)
	want := []string{"C:/Baz"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Remove mismatch: got %v want %v", got, want)
	}
}

func TestReplaceInstallRootPrefix(t *testing.T) {
	oldRoot := "C:/Users/me/.vfox/cache"
	newRoot := "D:/sdks"
	tests := []struct {
		name, value, want string
	}{
		{
			name:  "whole segment",
			value: oldRoot + "/alpha/v-1.0.0/bin",
			want:  newRoot + "/alpha/v-1.0.0/bin",
		},
		{
			name:  "semicolon path",
			value: "C:/Windows/system32;" + oldRoot + "/node/bin;C:/Git/cmd",
			want:  "C:/Windows/system32;" + newRoot + "/node/bin;C:/Git/cmd",
		},
		{
			name:  "case insensitive",
			value: "c:/users/me/.vfox/cache/Alpha",
			want:  newRoot + "/Alpha",
		},
		{
			name:  "substring is not a segment",
			value: oldRoot + "-backup/bin",
			want:  oldRoot + "-backup/bin",
		},
		{
			name:  "embedded text",
			value: "installed at " + oldRoot + "/bin today",
			want:  "installed at " + oldRoot + "/bin today",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReplaceInstallRootPrefix(tt.value, oldRoot, newRoot); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}

	oldWin := `C:\Users\me\.vfox\cache`
	newWin := `D:\sdks`
	value := `C:\Windows;` + oldWin + `\node\bin;` + oldWin + `-backup`
	want := `C:\Windows;` + newWin + `\node\bin;` + oldWin + `-backup`
	if got := ReplaceInstallRootPrefix(value, oldWin, newWin); got != want {
		t.Fatalf("backslash: got %q want %q", got, want)
	}

	uncNew := `\\server\share\sdks`
	uncValue := oldWin + `\node\bin`
	uncWant := uncNew + `\node\bin`
	if got := ReplaceInstallRootPrefix(uncValue, oldWin, uncNew); got != uncWant {
		t.Fatalf("unc destination: got %q want %q", got, uncWant)
	}
	uncOld := `\\server\share\vfox\cache`
	uncSrc := `C:\Windows;` + uncOld + `\node\bin;` + uncOld + `-backup`
	uncDst := `C:\Windows;` + uncNew + `\node\bin;` + uncOld + `-backup`
	if got := ReplaceInstallRootPrefix(uncSrc, uncOld, uncNew); got != uncDst {
		t.Fatalf("unc source: got %q want %q", got, uncDst)
	}
	// ".." stays under the share, the way filepath.Clean does on Windows.
	dotdotNew := `\\server\share\sdks\foo\..\..\..\bar`
	dotdotWant := `\\server\share\bar` + `\node\bin`
	if got := ReplaceInstallRootPrefix(uncValue, oldWin, dotdotNew); got != dotdotWant {
		t.Fatalf("unc dotdot: got %q want %q", got, dotdotWant)
	}
}

func TestSplitSemicolonSeparated(t *testing.T) {
	value := ";C:/Foo;;C:/Bar ;   C:/Baz;"

	got := splitSemicolonSeparated(value)
	want := []string{"C:/Foo", "C:/Bar", "C:/Baz"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Split mismatch: got %v want %v", got, want)
	}
}
