package string

import (
	"github.com/version-fox/vfox/internal/util"
	lua "github.com/yuin/gopher-lua"
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

	err := util.NewDecompressor(archiverPath).Decompress(targetPath)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	} else {
		L.Push(lua.LNil)
	}
	return 1
}
