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
	"strings"
	"testing"
)

func TestDetectIsAnOptionalHook(t *testing.T) {
	hook, ok := HookFuncMap["Detect"]
	if !ok || hook.Name != "Detect" || hook.Filename != "detect" || hook.Required {
		t.Errorf("Detect registration = %+v, found = %t", hook, ok)
	}
	// A plugin that does not implement it is still a valid plugin.
	if p := newTestPlugin(t, ""); p.HasFunction("Detect") {
		t.Error("the minimal plugin must not report hooks it does not have")
	}
}

func TestDetectHook(t *testing.T) {
	p := newTestPlugin(t, `
function PLUGIN:Detect(ctx)
  return {
    {version = "21.0.3-tem", path = "/usr/lib/jvm/temurin-21", note = "found in /usr/lib/jvm"},
    {version = "17-open", path = "/usr/lib/jvm/java-17-openjdk-amd64"},
  }
end`)
	items, err := p.Detect(&DetectHookCtx{})
	if err != nil {
		t.Fatal(err)
	}
	want := []*DetectHookResultItem{
		{Version: "21.0.3-tem", Path: "/usr/lib/jvm/temurin-21", Note: "found in /usr/lib/jvm"},
		{Version: "17-open", Path: "/usr/lib/jvm/java-17-openjdk-amd64"},
	}
	if !reflect.DeepEqual(items, want) {
		t.Errorf("items = %+v, want %+v", items, want)
	}
}

func TestDetectHookWithNothingFound(t *testing.T) {
	p := newTestPlugin(t, `function PLUGIN:Detect(ctx) return nil end`)
	if _, err := p.Detect(&DetectHookCtx{}); !errors.Is(err, ErrNoResultProvide) {
		t.Errorf("err = %v, want no result", err)
	}
	p = newTestPlugin(t, `function PLUGIN:Detect(ctx) return {} end`)
	items, err := p.Detect(&DetectHookCtx{})
	if err != nil || len(items) != 0 {
		t.Errorf("items = %v, err = %v, want an empty list", items, err)
	}
}

func TestDevelopmentRunsDetect(t *testing.T) {
	dir := t.TempDir()
	writePluginFile(t, dir, "main.lua", minimalPlugin+`
function PLUGIN:Detect(ctx) return {{version = "1", path = "/x"}} end
`)
	result, err := RunDevelopmentHook(context.Background(), dir, "Detect", nil, developmentOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := result.([]*DetectHookResultItem); !ok || len(got) != 1 || got[0].Path != "/x" {
		t.Errorf("Detect result = %#v", result)
	}
}

func TestDetectHookErrors(t *testing.T) {
	p := newTestPlugin(t, `function PLUGIN:Detect(ctx) error("boom") end`)
	if _, err := p.Detect(&DetectHookCtx{}); err == nil || errors.Is(err, ErrNoResultProvide) {
		t.Errorf("a failing hook must report its error, got %v", err)
	}

	p = newTestPlugin(t, `function PLUGIN:Detect(ctx) return "not a list" end`)
	if _, err := p.Detect(&DetectHookCtx{}); err == nil || errors.Is(err, ErrNoResultProvide) || !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("a result that is not a list must be a conversion error, got %v", err)
	}
}
