package env

import (
	"os"
	"runtime"
	"testing"
)

func TestPathFormat(t *testing.T) {
	if runtime.GOOS == "windows" {
		testdata := []struct {
			path string
			want string
		}{
			{
				path: "C:\\Program Files\\Git\\bin",
				want: "/c/Program Files/Git/bin",
			},
			{
				path: "D:\\b\\c",
				want: "/d/b/c",
			},
		}

		paths := NewPaths(EmptyPaths)
		for _, v := range testdata {
			paths.Add(v.path)
		}
		result := paths.Slice()
		for i, v := range testdata {
			if result[i] != v.path {
				t.Errorf("want: %s, got: %s", v.want, result[i])
			}
		}

		os.Setenv(HookFlag, "bash")
		paths = NewPaths(EmptyPaths)
		for _, v := range testdata {
			paths.Add(v.path)
		}
		result = paths.Slice()
		for i, v := range testdata {
			if result[i] != v.want {
				t.Errorf("want: %s, got: %s", v.want, result[i])
			}
		}

	} else {
		testdata := []struct {
			path string
			want string
		}{
			{
				path: "/bin/bash",
				want: "/bin/bash",
			},
			{
				path: "/usr/bin",
				want: "/usr/bin",
			},
		}

		paths := NewPaths(EmptyPaths)
		for _, v := range testdata {
			paths.Add(v.path)
		}
		result := paths.Slice()
		for i, v := range testdata {
			if result[i] != v.want {
				t.Errorf("want: %s, got: %s", v.want, result[i])
			}
		}
	}
}
