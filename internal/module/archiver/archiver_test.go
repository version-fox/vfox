package string

import (
	lua "github.com/yuin/gopher-lua"
	"os"
	"testing"
)

func TestDecompress(t *testing.T) {
	const str = `
	local archiver = require("vfox.archiver")
	local err = archiver.decompress("testdata/test.zip", "testdata/test")
	assert(err == nil, "strings.decompress()")
	local f = io.open("testdata/test/test.txt", "r")
	if f then 
		f:close()
	else
		error("file not found")
	end
	`
	defer func() {
		_ = os.RemoveAll("testdata/test")
	}()
	eval(str, t)
}

func eval(str string, t *testing.T) {
	s := lua.NewState()
	defer s.Close()

	Preload(s)
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}
