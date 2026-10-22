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
 */

package plugin

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestFingerprintIsAnOptionalHook(t *testing.T) {
	hook, ok := HookFuncMap["Fingerprint"]
	if !ok || hook.Name != "Fingerprint" || hook.Filename != "fingerprint" || hook.Required {
		t.Errorf("Fingerprint registration = %+v, found = %t", hook, ok)
	}
	// A plugin that does not implement it is still a valid plugin.
	if p := newTestPlugin(t, ""); p.HasFunction("Fingerprint") {
		t.Error("the minimal plugin must not report hooks it does not have")
	}
}

func TestFingerprintHook(t *testing.T) {
	p := newTestPlugin(t, `
function PLUGIN:Fingerprint(ctx)
  assert(ctx.path == "/sdk/flutter-3.47.4" and ctx.version == "3.47.4")
  assert(ctx.sdkInfo.example.path == "/sdk/flutter-3.47.4")
  -- numbers and booleans are facts too; they are compared as text
  return {facts = {head = "9584c67", build = 42, dirty = false}, ignored = "dropped"}
end`)
	result, err := p.Fingerprint(&FingerprintHookCtx{
		Path:    "/sdk/flutter-3.47.4",
		Version: "3.47.4",
		SdkInfo: map[string]*InstalledPackageItem{"example": {Name: "example", Version: "3.47.4", Path: "/sdk/flutter-3.47.4"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"head": "9584c67", "build": "42", "dirty": "false"}
	if !reflect.DeepEqual(result.Facts, want) {
		t.Errorf("facts = %v, want %v", result.Facts, want)
	}
}

func TestFingerprintHookWithNothingToSay(t *testing.T) {
	for name, body := range map[string]string{
		"nil":         `function PLUGIN:Fingerprint(ctx) return nil end`,
		"no facts":    `function PLUGIN:Fingerprint(ctx) return {} end`,
		"empty facts": `function PLUGIN:Fingerprint(ctx) return {facts = {}} end`,
		"flat table":  `function PLUGIN:Fingerprint(ctx) return {head = "x"} end`, // facts must be under "facts"
	} {
		t.Run(name, func(t *testing.T) {
			p := newTestPlugin(t, body)
			result, err := p.Fingerprint(&FingerprintHookCtx{})
			if !errors.Is(err, ErrNoResultProvide) || result != nil {
				t.Errorf("result = %v, err = %v, want no result", result, err)
			}
		})
	}
}

func TestFingerprintHookErrors(t *testing.T) {
	p := newTestPlugin(t, `function PLUGIN:Fingerprint(ctx) error("boom") end`)
	if _, err := p.Fingerprint(&FingerprintHookCtx{}); err == nil || errors.Is(err, ErrNoResultProvide) {
		t.Errorf("a failing hook must report its error, got %v", err)
	}

	p = newTestPlugin(t, `function PLUGIN:Fingerprint(ctx) return {facts = "not a table"} end`)
	if _, err := p.Fingerprint(&FingerprintHookCtx{}); err == nil || errors.Is(err, ErrNoResultProvide) {
		t.Errorf("facts that are not a table must be a conversion error, got %v", err)
	}

	p = newTestPlugin(t, `function PLUGIN:Fingerprint(ctx) return {facts = {nested = {1}}} end`)
	if _, err := p.Fingerprint(&FingerprintHookCtx{}); err == nil || errors.Is(err, ErrNoResultProvide) {
		t.Errorf("a fact that is a table must be a conversion error, got %v", err)
	}
}

func TestDevelopmentRunsFingerprint(t *testing.T) {
	dir := t.TempDir()
	writePluginFile(t, dir, "main.lua", minimalPlugin+`
function PLUGIN:Fingerprint(ctx) return {facts = {path = ctx.path}} end
`)
	result, err := RunDevelopmentHook(context.Background(), dir, "Fingerprint", []byte(`{"path":"/sdk","version":"1"}`), developmentOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := result.(*FingerprintHookResult); !ok || got.Facts["path"] != "/sdk" {
		t.Errorf("Fingerprint result = %#v", result)
	}

	// A plugin without the hook reports that, as for every other optional hook.
	plain := t.TempDir()
	writePluginFile(t, plain, "main.lua", minimalPlugin)
	if _, err := RunDevelopmentHook(context.Background(), plain, "Fingerprint", nil, developmentOptions()); err == nil {
		t.Error("running a hook the plugin does not implement must fail")
	}
}
