/*
 *    Copyright 2023 [lihan aooohan@gmail.com]
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

package html

import (
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
)

const luaFileTypeName = "file_operation"

type FileOperation struct {
	rootPath string
}

func (f *FileOperation) symlink(L *lua.LState) int {
	src := L.CheckString(1)
	dest := L.CheckString(2)
	// TODO check
	err := os.Symlink(filepath.Join(f.rootPath, src), filepath.Join(f.rootPath, dest))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	L.Push(lua.LTrue)
	return 1
}

func (f *FileOperation) luaMap() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"symlink": f.symlink,
	}
}

func (f *FileOperation) loader(L *lua.LState) int {
	t := L.NewTable()
	L.SetFuncs(t, f.luaMap())
	L.Push(t)
	return 1
}

func Preload(L *lua.LState, rootPath string) {
	operation := &FileOperation{rootPath: rootPath}
	L.PreloadModule("file", operation.loader)
}
