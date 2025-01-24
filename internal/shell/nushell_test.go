package shell

import (
	"bytes"
	"encoding/json"
	"github.com/version-fox/vfox/internal/env"
	"os"
	"reflect"
	"runtime"
	"slices"
	"testing"
	"text/template"
)

func TestActivate(t *testing.T) {
	var newline string
	if runtime.GOOS == "windows" {
		newline = "\r\n"
	} else {
		newline = "\n"
	}
	selfPath := "/path/to/vfox"
	want := newline +
		"# vfox configuration" + newline +
		"export-env {" + newline +
		"  def --env updateVfoxEnvironment [] {" + newline +
		"    let envData = (^'" + selfPath + "' env -s nushell | from json)" + newline +
		"    load-env $envData.envsToSet" + newline +
		"    hide-env ...$envData.envsToUnset" + newline +
		"  }" + newline +
		"  $env.config = ($env.config | upsert hooks.pre_prompt {" + newline +
		"    let currentValue = ($env.config | get -i hooks.pre_prompt)" + newline +
		"    if $currentValue == null {" + newline +
		"      [{updateVfoxEnvironment}]" + newline +
		"    } else {" + newline +
		"      $currentValue | append {updateVfoxEnvironment}" + newline +
		"    }" + newline +
		"  })" + newline +
		"  $env.__VFOX_SHELL = 'nushell'" + newline +
		"  $env.__VFOX_PID = $nu.pid" + newline +
		"  ^'" + selfPath + "' env --cleanup | ignore" + newline +
		"  updateVfoxEnvironment" + newline +
		"}" + newline

	n := nushell{}
	gotTemplate, err := n.Activate()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	parsedTemplate, err := template.New("activate").Parse(gotTemplate)
	if err != nil {
		t.Errorf("Unexpected error parsing template: %v", err)
		return
	}

	var buffer bytes.Buffer
	err = parsedTemplate.Execute(&buffer, struct{ SelfPath string }{selfPath})
	if err != nil {
		t.Errorf("Unexpected error executing template: %v", err)
		return
	}

	got := buffer.String()
	if got != want {
		t.Errorf("Output mismatch:\n\ngot=\n%v\n\nwant=\n%v", got, want)
	}
}

func TestExport(t *testing.T) {
	sep := string(os.PathListSeparator)

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
				EnvsToSet:   map[string]any{"PATH": []any{"/path1", "/path2"}},
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
				EnvsToSet:   map[string]any{"PATH": []any{"/path1", "/path2", "/path3"}, "FOO": "bar"},
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
