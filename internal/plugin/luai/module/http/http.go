/*
 *    Copyright 2025 Han Li and contributors
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

package http

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
	"github.com/version-fox/vfox/internal/base"
	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/plugin/luai/codec"
	lua "github.com/yuin/gopher-lua"
)

type Module struct {
	proxy  *config.Proxy
	client *http.Client
}

// Get performs a http get request
// @param url string
// @param headers table
// @return resp table
// @return err string
// local http = require("http")
//
//	http.get({
//	    url = "https://httpbin.org/json"
//	}) return (response, error)
//
//	response : {
//	    body = "",
//	    status_code = 200,
//	    headers = table
//	}
func (m *Module) Get(L *lua.LState) int {
	param := L.CheckTable(1)
	urlStr := param.RawGetString("url")
	if urlStr == lua.LNil {
		L.Push(lua.LNil)
		L.Push(lua.LString("url is required"))
		return 2
	}

	req, err := http.NewRequest("GET", urlStr.String(), nil)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	headersTable := param.RawGetString("headers")
	if headersTable != lua.LNil {
		if table, ok := headersTable.(*lua.LTable); ok {
			table.ForEach(func(key lua.LValue, value lua.LValue) {
				req.Header.Add(key.String(), value.String())
			})
		}
	}
	m.ensureUserAgent(L, req)

	resp, err := m.client.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()
	headers := L.NewTable()
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers.RawSetString(k, lua.LString(v[0]))
		}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	result := L.NewTable()
	L.SetField(result, "body", lua.LString(body))
	L.SetField(result, "status_code", lua.LNumber(resp.StatusCode))
	L.SetField(result, "headers", headers)
	L.SetField(result, "content_length", lua.LNumber(resp.ContentLength))
	L.Push(result)
	return 1
}

func (m *Module) Head(L *lua.LState) int {
	param := L.CheckTable(1)
	urlStr := param.RawGetString("url")
	if urlStr == lua.LNil {
		L.Push(lua.LNil)
		L.Push(lua.LString("url is required"))
		return 2
	}

	req, err := http.NewRequest("HEAD", urlStr.String(), nil)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	headersTable := param.RawGetString("headers")
	if headersTable != lua.LNil {
		if table, ok := headersTable.(*lua.LTable); ok {
			table.ForEach(func(key lua.LValue, value lua.LValue) {
				req.Header.Add(key.String(), value.String())
			})
		}
	}
	m.ensureUserAgent(L, req)

	resp, err := m.client.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	headers := L.NewTable()
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers.RawSetString(k, lua.LString(v[0]))
		}
	}
	result := L.NewTable()
	L.SetField(result, "status_code", lua.LNumber(resp.StatusCode))
	L.SetField(result, "headers", headers)
	L.SetField(result, "content_length", lua.LNumber(resp.ContentLength))
	L.Push(result)
	return 1
}

// DownloadFile performs a http get request to write stream to a file.
// @param url string
// @param headers table
// @return err string
// local http = require("http")
//
//		http.download_file({
//		    url = "http://ip.jsontest.com/"
//	     headers = {}
//		}, "/usr/path/file/") return error
func (m *Module) DownloadFile(L *lua.LState) int {
	param := L.CheckTable(1)
	fp := L.CheckString(2)
	if fp == "" {
		L.Push(lua.LString("filepath is required"))
		return 1
	}
	urlStr := param.RawGetString("url")
	if urlStr == lua.LNil {
		L.Push(lua.LString("url is required"))
		return 1
	}

	req, err := http.NewRequest("GET", urlStr.String(), nil)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	headersTable := param.RawGetString("headers")
	if headersTable != lua.LNil {
		if table, ok := headersTable.(*lua.LTable); ok {
			table.ForEach(func(key lua.LValue, value lua.LValue) {
				req.Header.Add(key.String(), value.String())
			})
		}
	}
	m.ensureUserAgent(L, req)
	resp, err := m.client.Do(req)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		L.Push(lua.LString("file not found"))
		return 1
	}
	out, err := os.Create(fp)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	defer out.Close()

	desc := "Downloading..."
	if filepath.Ext(urlStr.String()) != "" {
		desc = filepath.Base(urlStr.String())
	}

	bar := progressbar.NewOptions64(
		resp.ContentLength,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprintf(os.Stderr, "\n")
		}),
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
	defer bar.Close()
	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	return 0
}

func (m *Module) luaMap() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"get":           m.Get,
		"head":          m.Head,
		"download_file": m.DownloadFile,
	}
}

func createModule(proxy *config.Proxy) lua.LGFunction {
	return func(L *lua.LState) int {
		client := &http.Client{}
		if proxy.Enable {
			uri, err := url.Parse(proxy.Url)
			if err == nil {
				transPort := &http.Transport{
					Proxy: http.ProxyURL(uri),
				}
				client = &http.Client{
					Transport: transPort,
				}
			}
		}
		m := &Module{proxy: proxy, client: client}
		t := L.NewTable()
		L.SetFuncs(t, m.luaMap())
		L.Push(t)
		return 1
	}
}

func (m *Module) ensureUserAgent(L *lua.LState, req *http.Request) {
	if req.Header.Get("User-Agent") == "" {
		navigatorValue := L.GetGlobal(base.NavigatorObjKey)
		if navigatorValue != lua.LNil {
			var navigator base.Navigator
			err := codec.Unmarshal(navigatorValue, &navigator)
			if err == nil {
				req.Header.Set("User-Agent", navigator.UserAgent)
			}
		}
	}
}

func Preload(L *lua.LState, proxy *config.Proxy) {
	L.PreloadModule("http", createModule(proxy))
}
