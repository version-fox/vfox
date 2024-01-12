/*
 *    Copyright 2024 Han Li and contributors
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
	"testing"
)

func TestRequire(t *testing.T) {
	const str = `	
	local file = require("file")
	assert(type(file) == "table")
	assert(type(file.symlink) == "function")
	`
	evalLua(str, t)
}

func TestFind(t *testing.T) {
	const str = `	
	local file = require("file")
	file.symlink(src, dest)
	assert(div:text() == "hello world")
	`
	evalLua(str, t)
}

func TestFirst(t *testing.T) {
	const str = `	
	local html = require("html")
	local doc = html.parse("<html><body><div id='test'>123</div><div id='test'>456</div></body></html>")
	local div = doc:find("div")
	assert(div:first():text() == "123")
	`
	evalLua(str, t)
}

func TestContinuousFind(t *testing.T) {
	const str = `
	local html = require("html")
	local doc = html.parse("<html><body><div id='test'>test</div><div id='t2'>456</div></body></html>")
	local div = doc:find("body"):find("#t2")
	print(div:text() == "456")
	`
	evalLua(str, t)
}

func evalLua(str string, t *testing.T) {
	s := lua.NewState()
	defer s.Close()
	Preload(s, "")
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}

}
