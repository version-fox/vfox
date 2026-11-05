package archiver

import (
	"fmt"
	"path/filepath"

	lua "github.com/yuin/gopher-lua"

	"github.com/version-fox/vfox/internal/shared/util"
)

// Preload adds strings to the given Lua state's package.preload table. After it
// has been preloaded, it can be loaded using require:
//
//	local strings = require("vfox.archiver")
func Preload(L *lua.LState) {
	L.PreloadModule("vfox.archiver", Loader)
}

// Loader is the module loader function.
func Loader(L *lua.LState) int {
	t := L.NewTable()
	L.SetFuncs(t, api)
	L.Push(t)
	return 1
}

var api = map[string]lua.LGFunction{
	"decompress": decompress,
}

// decompress lua archiver.decompress(sourceFile, targetPath): port of go string.decompress() returns error
func decompress(L *lua.LState) int {
	archiverPath := L.CheckString(1)
	targetPath := L.CheckString(2)

	d := util.NewDecompressor(archiverPath)
	if d == nil {
		L.Push(lua.LString(fmt.Sprintf("unsupported archive type: %s", filepath.Ext(archiverPath))))
		return 1
	}
	err := d.Decompress(targetPath)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	} else {
		L.Push(lua.LNil)
	}
	return 1
}
