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
	"github.com/PuerkitoBio/goquery"
	lua "github.com/yuin/gopher-lua"
	"strings"
)

const luaHtmlDocumentTypeName = "html_document"
const luaSelectionTypeName = "html_selection"

func Preload(L *lua.LState) {
	L.PreloadModule("html", Loader)
}

// Loader is the module loader function.
func Loader(L *lua.LState) int {
	docMt := L.NewTypeMetatable(luaHtmlDocumentTypeName)
	L.SetField(docMt, "__index", L.SetFuncs(L.NewTable(), documentMethods))
	selectionMt := L.NewTypeMetatable(luaSelectionTypeName)
	L.SetField(selectionMt, "__index", L.SetFuncs(L.NewTable(), selectionMethods))
	table := L.NewTable()
	L.SetField(table, "parse", L.NewFunction(newHtmlDocument))
	L.Push(table)
	return 1
}

var documentMethods = map[string]lua.LGFunction{
	"find": documentFind,
}

var selectionMethods = map[string]lua.LGFunction{
	"text":  selectionText,
	"html":  selectionHtml,
	"find":  selectionFind,
	"first": selectionFirst,
	"each":  selectionEach,
	"attr":  selectionAttr,
	"eq":    selectionEq,
}

func selectionEq(L *lua.LState) int {
	s := checkSelection(L)
	idx := L.CheckInt(2)
	ud := L.NewUserData()
	ud.Value = s.Eq(idx)
	L.SetMetatable(ud, L.GetTypeMetatable(luaSelectionTypeName))
	L.Push(ud)
	return 1
}

func selectionAttr(L *lua.LState) int {
	s := checkSelection(L)
	attr := L.CheckString(2)
	ret, ok := s.Attr(attr)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(lua.LString(ret))
	return 1
}

func selectionEach(L *lua.LState) int {
	s := checkSelection(L)
	f := L.CheckFunction(2)
	s.Each(func(i int, selection *goquery.Selection) {
		ud := L.NewUserData()
		ud.Value = selection
		L.SetMetatable(ud, L.GetTypeMetatable(luaSelectionTypeName))
		err := L.CallByParam(lua.P{
			Fn:      f,
			NRet:    0,
			Protect: true,
		}, lua.LNumber(i+1), ud)
		if err != nil {
			L.RaiseError(err.Error())
		}
	})
	return 0
}

func newHtmlDocument(L *lua.LState) int {
	checkString := L.CheckString(1)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(checkString))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	doc.Text()
	ud := L.NewUserData()
	ud.Value = doc
	L.SetMetatable(ud, L.GetTypeMetatable(luaHtmlDocumentTypeName))
	L.Push(ud)
	return 1
}

func selectionFirst(state *lua.LState) int {
	s := checkSelection(state)
	ud := state.NewUserData()
	ud.Value = s.First()
	state.SetMetatable(ud, state.GetTypeMetatable(luaSelectionTypeName))
	state.Push(ud)
	return 1
}

func selectionFind(state *lua.LState) int {
	s := checkSelection(state)
	selector := state.CheckString(2)
	newV := s.Find(selector)
	ud := state.NewUserData()
	ud.Value = newV
	state.SetMetatable(ud, state.GetTypeMetatable(luaSelectionTypeName))
	state.Push(ud)
	return 1
}

func selectionHtml(state *lua.LState) int {
	s := checkSelection(state)
	ret, err := s.Html()
	if err != nil {
		state.RaiseError(err.Error())
		return 0
	}
	state.Push(lua.LString(ret))
	return 1
}

func selectionText(L *lua.LState) int {
	s := checkSelection(L)
	L.Push(lua.LString(s.Text()))
	return 1
}
func checkSelection(L *lua.LState) *goquery.Selection {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*goquery.Selection); ok {
		return v
	}
	L.ArgError(1, "selection expected")
	return nil
}
func documentFind(L *lua.LState) int {
	p := checkDocument(L)
	selector := L.CheckString(2)
	s := p.Find(selector)
	ud := L.NewUserData()
	ud.Value = s
	L.SetMetatable(ud, L.GetTypeMetatable(luaSelectionTypeName))
	L.Push(ud)
	return 1
}

func checkDocument(L *lua.LState) *goquery.Document {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*goquery.Document); ok {
		return v
	}
	L.ArgError(1, "document expected")
	return nil
}
