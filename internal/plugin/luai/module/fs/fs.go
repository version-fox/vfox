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

package fs

import (
	"os"
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
)

type Operation struct {
	rootPath string
}

func (f *Operation) path(path string) string {
	return filepath.Join(f.rootPath, path)
}

func raiseOnError(L *lua.LState, err error) {
	if err != nil {
		L.RaiseError("%s", err.Error())
	}
}

func returnTrue(L *lua.LState) int {
	L.Push(lua.LTrue)
	return 1
}

func (f *Operation) copy(L *lua.LState) int {
	src := L.CheckString(1)
	dest := L.CheckString(2)
	info, err := os.Stat(f.path(src))
	raiseOnError(L, err)
	if info.IsDir() {
		raiseOnError(L, os.CopyFS(f.path(dest), os.DirFS(f.path(src))))
		return returnTrue(L)
	}
	content, err := os.ReadFile(f.path(src))
	raiseOnError(L, err)
	raiseOnError(L, os.WriteFile(f.path(dest), content, info.Mode().Perm()))
	return returnTrue(L)
}

func (f *Operation) remove(L *lua.LState) int {
	path := L.CheckString(1)
	info, err := os.Lstat(f.path(path))
	raiseOnError(L, err)
	if info.IsDir() {
		raiseOnError(L, os.RemoveAll(f.path(path)))
	} else {
		raiseOnError(L, os.Remove(f.path(path)))
	}
	return returnTrue(L)
}

func (f *Operation) move(L *lua.LState) int {
	src := L.CheckString(1)
	dest := L.CheckString(2)
	srcPath := f.path(src)
	destPath := f.path(dest)
	if info, err := os.Stat(destPath); err == nil && info.IsDir() {
		destPath = filepath.Join(destPath, filepath.Base(srcPath))
	}
	raiseOnError(L, os.Rename(srcPath, destPath))
	return returnTrue(L)
}

func (f *Operation) symlink(L *lua.LState) int {
	src := L.CheckString(1)
	dest := L.CheckString(2)
	raiseOnError(L, os.Symlink(f.path(src), f.path(dest)))
	return returnTrue(L)
}

func (f *Operation) luaMap() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"copy":    f.copy,
		"remove":  f.remove,
		"move":    f.move,
		"symlink": f.symlink,
	}
}

func (f *Operation) loader(L *lua.LState) int {
	t := L.NewTable()
	L.SetFuncs(t, f.luaMap())
	L.Push(t)
	return 1
}

func Preload(L *lua.LState, rootPath string) {
	operation := &Operation{rootPath: rootPath}
	L.PreloadModule("fs", operation.loader)
}
