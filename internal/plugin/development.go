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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	lua "github.com/yuin/gopher-lua"

	"github.com/version-fox/vfox/internal/plugin/luai"
	"github.com/version-fox/vfox/internal/plugin/luai/codec"
	"github.com/version-fox/vfox/internal/plugin/luai/module"
	"github.com/version-fox/vfox/internal/shared/util"
)

// DevelopmentOptions affect only the temporary VM, never installed plugins or
// user/project configuration. The orchestration layer supplies RuntimeVersion.
type DevelopmentOptions struct {
	RuntimeVersion string
	OS, Arch       string
	Online         bool
	Output         io.Writer
}

type developmentVM struct {
	vm            *luai.LuaVM
	hookThread    *lua.LState
	cancelHooks   context.CancelFunc
	transport     *developmentTransport
	runtime       RuntimeInfo
	environment   map[string]lua.LValue
	plugin        *LuaPlugin
	loadAttempted bool
}

func newDevelopmentVM(ctx context.Context, directory string, options DevelopmentOptions, testing bool) (*developmentVM, error) {
	if options.Output == nil {
		options.Output = os.Stderr
	}
	if options.OS == "" {
		options.OS = string(util.GetOSType())
	}
	if options.Arch == "" {
		options.Arch = string(util.GetArchType())
	}
	d := &developmentVM{
		vm:          luai.NewLuaVM(),
		transport:   &developmentTransport{online: options.Online},
		runtime:     RuntimeInfo{OsType: options.OS, ArchType: options.Arch, Version: options.RuntimeVersion, PluginDirPath: directory},
		environment: make(map[string]lua.LValue),
	}
	d.vm.Instance.SetContext(ctx)
	if err := d.vm.Prepare(&module.PreloadOptions{
		Output: options.Output,
		HTTPTransport: func(fallback http.RoundTripper) http.RoundTripper {
			d.transport.fallback = fallback
			return d.transport
		},
	}); err != nil {
		d.close()
		return nil, err
	}
	L := d.vm.Instance
	configureDevelopmentOutput(L, options.Output, testing)
	if testing {
		// Use a separate call stack in the SAME VM (shared globals/modules).
		// gopher-lua #448 closes the caller's open upvalues on a hook error
		// when a nested protected call uses the test's stack.
		d.hookThread, d.cancelHooks = L.NewThread()
		L.GetGlobal("os").(*lua.LTable).RawSetString("getenv", L.NewFunction(func(L *lua.LState) int {
			value, exists := d.environment[L.CheckString(1)]
			if !exists || value == lua.LFalse {
				value = lua.LNil
			}
			L.Push(value)
			return 1
		}))
		L.PreloadModule("vfox.test", func(L *lua.LState) int {
			api := L.NewTable()
			api.RawSetString("load", L.NewFunction(d.loadForTest))
			L.Push(api)
			return 1
		})
	}
	return d, nil
}

func (d *developmentVM) close() {
	if d.cancelHooks != nil {
		d.cancelHooks()
	}
	d.vm.Close()
	d.transport.close()
}

func (d *developmentVM) load() error {
	original := d.vm.Instance
	if d.hookThread != nil {
		d.vm.Instance = d.hookThread
	}
	defer func() { d.vm.Instance = original }()
	p, metadata, err := loadLuaPlugin(d.vm, d.runtime.PluginDirPath, d.runtime)
	if err != nil {
		return err
	}
	wrapper := &Wrapper{Metadata: metadata, Plugin: p, InstalledPath: d.runtime.PluginDirPath}
	if err := wrapper.validate(); err != nil {
		return err
	}
	if metadata.MinRuntimeVersion != "" && util.CompareVersion(metadata.MinRuntimeVersion, d.runtime.Version) > 0 {
		return fmt.Errorf("plugin requires vfox >= %s", metadata.MinRuntimeVersion)
	}
	p.development = true
	d.plugin = p
	return nil
}

func (d *developmentVM) loadForTest(L *lua.LState) int {
	if d.loadAttempted {
		L.RaiseError("test.load may only be called once per test file")
	}
	d.loadAttempted = true
	options := L.OptTable(1, L.NewTable())
	options.ForEach(func(key, value lua.LValue) {
		switch key.String() {
		case "os", "arch":
			text, ok := value.(lua.LString)
			if !ok || text == "" {
				L.RaiseError("test.load %s must be a nonempty string", key)
			}
			if key.String() == "os" {
				d.runtime.OsType = string(text)
			} else {
				d.runtime.ArchType = string(text)
			}
		case "env":
			env, ok := value.(*lua.LTable)
			if !ok {
				L.RaiseError("test.load env must be a table")
			}
			env.ForEach(func(key, value lua.LValue) {
				if key.Type() != lua.LTString || (value.Type() != lua.LTString && value != lua.LFalse) {
					L.RaiseError("test.load env requires string keys and string or false values")
				}
				d.environment[key.String()] = value
			})
		case "http":
			fn, ok := value.(*lua.LFunction)
			if !ok {
				L.RaiseError("test.load http must be a function")
			}
			d.transport.handler = fn
		default:
			L.RaiseError("unknown test.load option %s", key)
		}
	})
	if err := d.load(); err != nil {
		L.RaiseError("load plugin: %s", err)
	}
	proxy := L.NewTable()
	for name := range HookFuncMap {
		proxy.RawSetString(name, L.NewFunction(func(L *lua.LState) int {
			if L.Get(1) != proxy {
				raiseDevelopmentError(L, fmt.Errorf("use plugin:%s(ctx) with a context table", name))
			}
			input, ok := L.Get(2).(*lua.LTable)
			if !ok {
				raiseDevelopmentError(L, fmt.Errorf("%s context must be a table", name))
			}
			original := d.vm.Instance
			d.vm.Instance = d.hookThread
			defer func() { d.vm.Instance = original }()
			result, err := invokeDevelopmentHook(d.plugin, name, func(target any) error { return codec.Unmarshal(input, target) })
			if err != nil {
				raiseDevelopmentError(L, fmt.Errorf("%s: %w", name, err))
			}
			value, err := codec.Marshal(L, result)
			if err != nil {
				raiseDevelopmentError(L, fmt.Errorf("%s result: %w", name, err))
			}
			L.Push(value)
			return 1
		}))
	}
	L.Push(proxy)
	return 1
}

// Raise an ordinary Lua error without closing the test caller's upvalues.
// L.RaiseError would close them even when Lua pcall catches the bridge error:
// https://github.com/yuin/gopher-lua/issues/448.
func raiseDevelopmentError(L *lua.LState, err error) {
	panic(&lua.ApiError{Type: lua.ApiErrorRun, Object: lua.LString(L.Where(1) + err.Error())})
}

// TestDevelopmentFile runs ordinary Lua assertions in a single fresh VM.
func TestDevelopmentFile(ctx context.Context, directory, file string, options DevelopmentOptions) error {
	d, err := newDevelopmentVM(ctx, directory, options, true)
	if err != nil {
		return err
	}
	defer d.close()
	if err := d.vm.Instance.DoFile(file); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if d.plugin == nil {
		return fmt.Errorf("test file did not load a plugin with test.load()")
	}
	return nil
}

// RunDevelopmentHook calls exactly one hook, without SDK lifecycle orchestration.
func RunDevelopmentHook(ctx context.Context, directory, hook string, input []byte, options DevelopmentOptions) (any, error) {
	if _, ok := HookFuncMap[hook]; !ok {
		return nil, fmt.Errorf("unknown hook %q", hook)
	}
	input = bytes.TrimSpace(input)
	if len(input) == 0 {
		input = []byte("{}")
	}
	if input[0] != '{' || !json.Valid(input) {
		return nil, fmt.Errorf("hook input must be a JSON object")
	}
	d, err := newDevelopmentVM(ctx, directory, options, false)
	if err != nil {
		return nil, err
	}
	defer d.close()
	if err := d.load(); err != nil {
		return nil, err
	}
	result, err := invokeDevelopmentHook(d.plugin, hook, func(target any) error { return json.Unmarshal(input, target) })
	if err != nil {
		return nil, fmt.Errorf("%s: %w", hook, err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func callDevelopmentHook[C, R any](decode func(any) error, call func(*C) (R, error)) (any, error) {
	var input C
	if err := decode(&input); err != nil {
		return nil, fmt.Errorf("hook input: %w", err)
	}
	result, err := call(&input)
	if errors.Is(err, ErrNoResultProvide) {
		return nil, nil
	}
	return result, err
}

type developmentLegacyContext struct {
	Filepath          string   `json:"filepath"`
	Filename          string   `json:"filename"`
	Strategy          string   `json:"strategy"`
	InstalledVersions []string `json:"installedVersions"`
}

func invokeDevelopmentHook(p Plugin, name string, decode func(any) error) (any, error) {
	if !p.HasFunction(name) {
		return nil, fmt.Errorf("[%s] function not found", name)
	}
	switch name {
	case "Available":
		return callDevelopmentHook(decode, p.Available)
	case "PreInstall":
		return callDevelopmentHook(decode, p.PreInstall)
	case "EnvKeys":
		return callDevelopmentHook(decode, p.EnvKeys)
	case "PreUse":
		return callDevelopmentHook(decode, p.PreUse)
	case "PostInstall":
		return callDevelopmentHook(decode, func(ctx *PostInstallHookCtx) (any, error) { return nil, p.PostInstall(ctx) })
	case "PreUninstall":
		return callDevelopmentHook(decode, func(ctx *PreUninstallHookCtx) (any, error) { return nil, p.PreUninstall(ctx) })
	case "ParseLegacyFile":
		return callDevelopmentHook(decode, func(ctx *developmentLegacyContext) (*ParseLegacyFileResult, error) {
			return p.ParseLegacyFile(&ParseLegacyFileHookCtx{
				Filepath: ctx.Filepath, Filename: ctx.Filename, Strategy: ctx.Strategy,
				GetInstalledVersions: func() []string { return ctx.InstalledVersions },
			})
		})
	default:
		return nil, fmt.Errorf("unknown hook %q", name)
	}
}
