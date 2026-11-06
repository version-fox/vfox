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

package http

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/plugin/luai/codec"
	"github.com/version-fox/vfox/internal/shared/util"

	"github.com/version-fox/vfox/internal/config"
	lua "github.com/yuin/gopher-lua"
)

const jsonUrl = `https://version-fox.github.io/vfox-plugins/index.json`

func TestWithConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skip on windows, the proxy won't error on windows.")
	}

	const str = `
	local http = require("http")
	assert(type(http) == "table")
	assert(type(http.get) == "function")
	local resp, err = http.get({
        url = jsonUrl
    })
	print(err)
	assert(err == 'Get "'.. jsonUrl .. '": proxyconnect tcp: dial tcp 127.0.0.1:80: connect: connection refused')
	`
	s := lua.NewState()
	defer s.Close()

	s.SetGlobal("jsonUrl", lua.LString(jsonUrl))

	Preload(s, &config.Proxy{
		Enable: true,
		Url:    "http://127.0.0.1",
	}, nil)

	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}
func TestGetRequest(t *testing.T) {
	const str = `
	local http = require("http")
	assert(type(http) == "table")
	assert(type(http.get) == "function")
	local resp, err = http.get({
        url = jsonUrl
    })
	assert(err == nil)
	assert(resp.status_code == 200)
	assert(resp.headers['Content-Type'] == 'application/json; charset=utf-8')
	`
	eval(str, t)
}

func TestHeadRequest(t *testing.T) {
	const str = `
	local http = require("http")
	assert(type(http) == "table")
	assert(type(http.get) == "function")
	local resp, err = http.head({
        url = jsonUrl
    })
	assert(err == nil)
	assert(resp.status_code == 200)
	assert(resp.content_length ~= 0)
	`
	eval(str, t)
}

func TestDownloadFile(t *testing.T) {
	const body = "index bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	indexPath := filepath.ToSlash(filepath.Join(t.TempDir(), "index.json"))
	missingPath := filepath.ToSlash(filepath.Join(t.TempDir(), "xxx.json"))
	str := fmt.Sprintf(`
	local http = require("http")
	assert(type(http) == "table")
	assert(type(http.get) == "function")
	local err = http.download_file({ url = %q }, %q)
	assert(err == nil, tostring(err))
	local err = http.download_file({ url = %q }, %q)
	assert(err == "file not found", tostring(err))
	`, srv.URL+"/index.json", indexPath, srv.URL+"/xxx.json", missingPath)
	eval(str, t)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(data) != body {
		t.Fatalf("downloaded content = %q, want %q", data, body)
	}
	if util.FileExists(missingPath) {
		t.Error("a 404 must not create the target file")
	}
}

func TestDownloadFileRejectsUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>refused</html>"))
	}))
	defer srv.Close()

	refusedPath := filepath.ToSlash(filepath.Join(t.TempDir(), "refused.zip"))
	str := fmt.Sprintf(`
	local http = require("http")
	local err = http.download_file({ url = %q }, %q)
	assert(err ~= nil, "must return an error")
	assert(string.find(err, "unexpected HTTP status 403 Forbidden", 1, true) ~= nil, tostring(err))
	assert(io.open(%q) == nil, "no file must be created")
	`, srv.URL+"/refused.zip", refusedPath, refusedPath)
	eval(str, t)
	if util.FileExists(refusedPath) {
		t.Error("a refused download must not create the target file")
	}
}

func eval(str string, t *testing.T) {
	s := lua.NewState()
	defer s.Close()

	s.SetGlobal("jsonUrl", lua.LString(jsonUrl))
	Preload(s, config.EmptyProxy, nil)

	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}

// The errors returned to a plugin must not echo the credentials or the query
// tokens of its URL, whether the request fails to parse or to connect.
func TestURLErrorsAreRedacted(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		secrets []string
	}{
		{"connection refused", "http://user:s3cret@127.0.0.1:1/file.zip?token=abc123", []string{"s3cret", "abc123"}},
		{"rejected by url.Parse", `http://user:Zq9%ss@127.0.0.1/file.zip`, []string{"Zq9", "%ss"}},
	}
	for _, tc := range cases {
		for _, method := range []string{"get", "head", "download_file"} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				s := lua.NewState()
				defer s.Close()
				Preload(s, config.EmptyProxy, nil)
				s.SetGlobal("url", lua.LString(tc.url))
				s.SetGlobal("dest", lua.LString(filepath.Join(t.TempDir(), "out")))
				script := fmt.Sprintf(`local http = require("http")
local resp, err = http.%s({url = url})
assert(resp == nil)
assert(err ~= nil)
errMsg = err`, method)
				if method == "download_file" {
					script = `local http = require("http")
local err = http.download_file({url = url}, dest)
assert(err ~= nil)
errMsg = err`
				}
				if err := s.DoString(script); err != nil {
					t.Fatal(err)
				}
				msg := s.GetGlobal("errMsg").String()
				for _, secret := range tc.secrets {
					if strings.Contains(msg, secret) {
						t.Errorf("error = %q, want %q redacted", msg, secret)
					}
				}
			})
		}
	}
}

// The progress bar label is the base of the URL path, so a query token is not
// printed while a download runs.
func TestDownloadFileLabelUsesURLPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("archive bytes"))
	}))
	defer srv.Close()

	var out bytes.Buffer
	s := lua.NewState()
	defer s.Close()
	PreloadWithOptions(s, config.EmptyProxy, nil, Options{Output: &out})
	s.SetGlobal("url", lua.LString(srv.URL+"/file.zip?token=abc123"))
	s.SetGlobal("dest", lua.LString(filepath.Join(t.TempDir(), "file.zip")))
	if err := s.DoString(`local http = require("http"); assert(http.download_file({url = url}, dest) == nil)`); err != nil {
		t.Fatal(err)
	}
	printed := out.String()
	if !strings.Contains(printed, "file.zip") {
		t.Fatalf("progress output = %q, want the file name as the label", printed)
	}
	if strings.Contains(printed, "abc123") {
		t.Fatalf("progress output = %q, want the query token redacted", printed)
	}
}

func setUserAgent(L *lua.LState, ua string) {
	L.SetGlobal(codec.NavigatorObjKey, codec.MustMarshal(L, codec.Navigator{UserAgent: ua}))
}

func TestUserAgentDefault(t *testing.T) {
	uaCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uaCh <- r.UserAgent()
		_, _ = w.Write([]byte(`ok`))
	}))
	defer server.Close()

	ls := lua.NewState()
	ua := "vfox/0.7.0 vfox-nodejs/0.3.0"
	setUserAgent(ls, ua)
	defer ls.Close()
	Preload(ls, config.EmptyProxy, nil)

	script := fmt.Sprintf(`
	local http = require("http")
	local resp, err = http.get({
	    url = "%s"
	})
	assert(err == nil)
	assert(resp.status_code == 200)
	`, server.URL)
	if err := ls.DoString(script); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case ua := <-uaCh:
		expected := "vfox/0.7.0 vfox-nodejs/0.3.0"
		if ua != expected {
			t.Fatalf("expected user-agent %q, got %q", expected, ua)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for request")
	}
}

func TestUserAgentAppend(t *testing.T) {
	uaCh := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uaCh <- r.UserAgent()
		_, _ = w.Write([]byte(`ok`))
	}))
	defer server.Close()

	ls := lua.NewState()
	ua := "vfox/0.7.0 vfox-nodejs/0.3.0"
	setUserAgent(ls, ua)
	defer ls.Close()

	Preload(ls, config.EmptyProxy, nil)

	script := fmt.Sprintf(`
	local http = require("http")
	local resp, err = http.get({
	    url = "%s",
	    headers = {
	        ["User-Agent"] = "custom-agent"
	    }
	})
	assert(err == nil)
	assert(resp.status_code == 200)
	`, server.URL)
	if err := ls.DoString(script); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case ua := <-uaCh:
		expected := "custom-agent"
		if ua != expected {
			t.Fatalf("expected user-agent %q, got %q", expected, ua)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for request")
	}
}

// Exercise the same clients used by Lua, with short deadlines for local tests.
func TestRequestTimeouts(t *testing.T) {
	for _, proxyEnabled := range []bool{false, true} {
		for _, method := range []string{"get", "head", "download_file"} {
			for _, stallBody := range []bool{false, true} {
				if method == "head" && stallBody {
					continue
				}
				t.Run(fmt.Sprintf("proxy=%t/%s/body=%t", proxyEnabled, method, stallBody), func(t *testing.T) {
					t.Parallel()
					release := make(chan struct{})
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if stallBody {
							w.Header().Set("Content-Length", "100")
							w.WriteHeader(http.StatusOK)
							w.(http.Flusher).Flush()
						}
						select {
						case <-r.Context().Done():
						case <-release:
						case <-time.After(2 * time.Second):
						}
					}))
					defer server.Close()
					defer close(release)
					m := newModule(&config.Proxy{Enable: proxyEnabled, Url: server.URL}, nil)
					if m.client.Timeout != 30*time.Second || m.downloadClient.Timeout != 30*time.Minute {
						t.Fatal("unexpected default request/download deadlines")
					}
					m.client.Timeout = 100 * time.Millisecond
					m.downloadClient.Timeout = 100 * time.Millisecond
					ls := lua.NewState()
					defer ls.Close()
					ls.SetGlobal("http", ls.SetFuncs(ls.NewTable(), m.luaMap()))
					url := server.URL
					if proxyEnabled {
						url = "http://vfox-timeout.invalid/manifest"
					}
					ls.SetGlobal("url", lua.LString(url))
					ls.SetGlobal("destination", lua.LString(filepath.Join(t.TempDir(), "download")))
					script := fmt.Sprintf(`local resp, err = http.%s({url = url}); assert(resp == nil); assert(string.find(err, "deadline exceeded") or string.find(err, "timeout"), err)`, method)
					if method == "download_file" {
						script = `local err = http.download_file({url = url}, destination); assert(string.find(err, "deadline exceeded") or string.find(err, "timeout"), err)`
					}
					if err := ls.DoString(script); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestDownloadUsesSeparateDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("complete download"))
	}))
	defer server.Close()
	m := newModule(config.EmptyProxy, nil)
	m.client.Timeout = time.Millisecond
	m.downloadClient.Timeout = time.Second
	ls := lua.NewState()
	defer ls.Close()
	ls.SetGlobal("http", ls.SetFuncs(ls.NewTable(), m.luaMap()))
	ls.SetGlobal("url", lua.LString(server.URL))
	destination := filepath.Join(t.TempDir(), "download")
	ls.SetGlobal("destination", lua.LString(destination))
	if err := ls.DoString(`assert(http.download_file({url = url}, destination) == nil)`); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(destination)
	if err != nil || string(body) != "complete download" {
		t.Fatalf("download = %q, err = %v", body, err)
	}
}
