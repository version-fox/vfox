package shell

import (
	"encoding/json"
	"os"
	"reflect"
	"runtime"
	"slices"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

func TestExport(t *testing.T) {
	sep := string(os.PathListSeparator)

	var pathVarName string
	if runtime.GOOS == "windows" {
		pathVarName = "Path"
	} else {
		pathVarName = "PATH"
	}

	tests := []struct {
		name string
		envs env.Vars
		want nushellExportData
	}{
		{
			"Empty",
			env.Vars{},
			nushellExportData{
				EnvsToSet:   make(map[string]any),
				EnvsToUnset: make([]string, 0)},
		},
		{
			"SingleEnv",
			env.Vars{"FOO": newString("bar")},
			nushellExportData{
				EnvsToSet:   map[string]any{"FOO": "bar"},
				EnvsToUnset: make([]string, 0),
			},
		},
		{
			"MultipleEnvs",
			env.Vars{"FOO": newString("bar"), "BAZ": newString("qux")},
			nushellExportData{
				EnvsToSet:   map[string]any{"FOO": "bar", "BAZ": "qux"},
				EnvsToUnset: make([]string, 0),
			},
		},
		{
			"UnsetEnv",
			env.Vars{"FOO": nil},
			nushellExportData{
				EnvsToSet:   make(map[string]any),
				EnvsToUnset: []string{"FOO"},
			},
		},
		{
			"MixedEnvs",
			env.Vars{"FOO": newString("bar"), "BAZ": nil},
			nushellExportData{
				EnvsToSet:   map[string]any{"FOO": "bar"},
				EnvsToUnset: []string{"BAZ"},
			},
		},
		{
			"MultipleUnsetEnvs",
			env.Vars{"FOO": nil, "BAZ": nil},
			nushellExportData{
				EnvsToSet:   make(map[string]any),
				EnvsToUnset: []string{"FOO", "BAZ"},
			},
		},
		{
			"PathEnv",
			env.Vars{"PATH": newString("/path1" + sep + "/path2")},
			nushellExportData{
				EnvsToSet:   map[string]any{pathVarName: []any{"/path1", "/path2"}},
				EnvsToUnset: make([]string, 0),
			},
		},
		{
			"PathAndOtherEnv",
			env.Vars{
				"PATH": newString("/path1" + sep + "/path2" + sep + "/path3"),
				"FOO":  newString("bar"),
				"BAZ":  nil,
			},
			nushellExportData{
				EnvsToSet:   map[string]any{pathVarName: []any{"/path1", "/path2", "/path3"}, "FOO": "bar"},
				EnvsToUnset: []string{"BAZ"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runExportTest(t, test.envs, test.want)
		})
	}
}

func runExportTest(t *testing.T, envs env.Vars, want nushellExportData) {
	n := nushell{}
	got := n.Export(envs)
	var gotData nushellExportData
	err := json.Unmarshal([]byte(got), &gotData)
	if err != nil {
		t.Errorf("%s: error unmarshaling export data - %v", t.Name(), err)
		return
	}

	slices.Sort(want.EnvsToUnset)
	slices.Sort(gotData.EnvsToUnset)
	if !reflect.DeepEqual(gotData, want) {
		t.Errorf("%s: export data mismatch - want %v, got %v", t.Name(), want, gotData)
	}
}

func newString(s string) *string {
	return &s
}
