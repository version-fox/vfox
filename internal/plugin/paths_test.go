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
 *
 */

package plugin

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestPathsIsAnOptionalHook(t *testing.T) {
	hook, ok := HookFuncMap["Paths"]
	if !ok || hook.Name != "Paths" || hook.Filename != "paths" || hook.Required {
		t.Errorf("Paths registration = %+v, found = %t", hook, ok)
	}
	// A plugin that does not implement it is still a valid plugin.
	if p := newTestPlugin(t, ""); p.HasFunction("Paths") {
		t.Error("the minimal plugin must not report hooks it does not have")
	}
}

func TestPathsHook(t *testing.T) {
	p := newTestPlugin(t, `
function PLUGIN:Paths(ctx)
  assert(ctx.path == "/sdk/python-3.12.3" and ctx.version == "3.12.3")
  assert(ctx.sdkInfo.example.path == "/sdk/python-3.12.3")
  -- values are compared as text; keys outside paths are dropped
  return {paths = {include = ctx.path .. "/include", lib = ctx.path .. "/lib", extra = 1}, ignored = "dropped"}
end`)
	result, err := p.Paths(&PathsHookCtx{
		Path:    "/sdk/python-3.12.3",
		Version: "3.12.3",
		SdkInfo: map[string]*InstalledPackageItem{"example": {Name: "example", Version: "3.12.3", Path: "/sdk/python-3.12.3"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"include": "/sdk/python-3.12.3/include", "lib": "/sdk/python-3.12.3/lib", "extra": "1"}
	if !reflect.DeepEqual(result.Paths, want) {
		t.Errorf("paths = %v, want %v", result.Paths, want)
	}
}

func TestPathsHookWithNothingToSay(t *testing.T) {
	for name, body := range map[string]string{
		"nil":         `function PLUGIN:Paths(ctx) return nil end`,
		"no paths":    `function PLUGIN:Paths(ctx) return {} end`,
		"empty paths": `function PLUGIN:Paths(ctx) return {paths = {}} end`,
		"flat table":  `function PLUGIN:Paths(ctx) return {include = "/x"} end`, // directories must be under "paths"
	} {
		t.Run(name, func(t *testing.T) {
			p := newTestPlugin(t, body)
			result, err := p.Paths(&PathsHookCtx{})
			if !errors.Is(err, ErrNoResultProvide) || result != nil {
				t.Errorf("result = %v, err = %v, want no result", result, err)
			}
		})
	}
}

func TestPathsHookErrors(t *testing.T) {
	p := newTestPlugin(t, `function PLUGIN:Paths(ctx) error("boom") end`)
	if _, err := p.Paths(&PathsHookCtx{}); err == nil || errors.Is(err, ErrNoResultProvide) {
		t.Errorf("a failing hook must report its error, got %v", err)
	}

	p = newTestPlugin(t, `function PLUGIN:Paths(ctx) return {paths = "not a table"} end`)
	if _, err := p.Paths(&PathsHookCtx{}); err == nil || errors.Is(err, ErrNoResultProvide) {
		t.Errorf("paths that are not a table must be a conversion error, got %v", err)
	}

	p = newTestPlugin(t, `function PLUGIN:Paths(ctx) return {paths = {include = {"/x"}}} end`)
	if _, err := p.Paths(&PathsHookCtx{}); err == nil || errors.Is(err, ErrNoResultProvide) {
		t.Errorf("a directory that is a table must be a conversion error, got %v", err)
	}
}

func TestDevelopmentRunsPaths(t *testing.T) {
	dir := t.TempDir()
	writePluginFile(t, dir, "main.lua", minimalPlugin+`
function PLUGIN:Paths(ctx) return {paths = {include = ctx.path .. "/include"}} end
`)
	result, err := RunDevelopmentHook(context.Background(), dir, "Paths", []byte(`{"path":"/sdk","version":"1"}`), developmentOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := result.(*PathsHookResult); !ok || got.Paths["include"] != "/sdk/include" {
		t.Errorf("Paths result = %#v", result)
	}

	// A plugin without the hook reports that, as for every other optional hook.
	plain := t.TempDir()
	writePluginFile(t, plain, "main.lua", minimalPlugin)
	if _, err := RunDevelopmentHook(context.Background(), plain, "Paths", nil, developmentOptions()); err == nil {
		t.Error("running a hook the plugin does not implement must fail")
	}
}
