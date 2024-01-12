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
	local html = require("html")
	assert(type(html) == "table")
	assert(type(html.parse) == "function")
	`
	evalLua(str, t)
}

func TestFind(t *testing.T) {
	const str = `	
	local html = require("html")
	local doc = html.parse("<html><body><div id='test'>hello world</div></body></html>")
	local div = doc:find("#test")
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
	assert(div:text() == "456")
	`
	evalLua(str, t)
}

func TestEach(t *testing.T) {
	const str = `	
	local html = require("html")
	local doc = html.parse("<html><body><div id='test'>hello world</div><div>aabbb</div></body></html>")
	doc:find("div"):each(function(i, selection)
		if i == 1 then
			assert(selection:text() == "hello world")
		elseif i == 2 then
			assert(selection:text() == "aabbb")
		end
	end)
	`
	evalLua(str, t)
}

func TestAttr(t *testing.T) {
	const str = `
	local html = require("html")
	local doc = html.parse("<html><body><div id='t2' name='123'>456</div></body></html>")
	local div = doc:find("#t2")
	assert(div:attr("name") == "123")	
	assert(div:attr("test") == nil)
	`
	evalLua(str, t)
}

func TestEq(t *testing.T) {
	const str = `
	local html = require("html")
	local doc = html.parse("<html><body><div id='t2' name='123'>456</div><div>222</div></body></html>")
	local s = doc:find("div"):eq(1)
	local f = doc:find("div"):eq(0)
	local ss = doc:find("div"):eq(2)
	print(ss:text() == "")
	assert(s:text() == "222")	
	assert(f:text() == "456")	
	`
	evalLua(str, t)
}

func evalLua(str string, t *testing.T) {
	s := lua.NewState()
	defer s.Close()
	Preload(s)
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}

}
