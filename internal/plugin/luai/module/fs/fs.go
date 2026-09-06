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
	"io"
	"io/fs"
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
		raiseOnError(L, copyDirectory(f.path(src), f.path(dest)))
		return returnTrue(L)
	}
	content, err := os.ReadFile(f.path(src))
	raiseOnError(L, err)
	raiseOnError(L, os.WriteFile(f.path(dest), content, info.Mode().Perm()))
	return returnTrue(L)
}

// copyDirectory preserves symlinks without following them, including on Go 1.24.
// Like os.CopyFS, it does not overwrite existing files.
func copyDirectory(src, dest string) error {
	// The source itself may name a directory through a symlink.
	src, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0777)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if !entry.Type().IsRegular() {
			return &os.PathError{Op: "copy", Path: path, Err: os.ErrInvalid}
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666|info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
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
	srcPath, err := filepath.Abs(f.path(src))
	raiseOnError(L, err)
	raiseOnError(L, os.Symlink(srcPath, f.path(dest)))
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
