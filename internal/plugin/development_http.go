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

package plugin

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"github.com/version-fox/vfox/internal/plugin/luai/codec"
	luahttp "github.com/version-fox/vfox/internal/plugin/luai/module/http"
)

// developmentTransport is confined to one VM. RoundTrip never starts Lua work
// on another goroutine; the standard HTTP module calls it synchronously.
type developmentTransport struct {
	online   bool
	handler  *lua.LFunction
	fallback http.RoundTripper
}

func (t *developmentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	if t.handler == nil {
		if !t.online {
			return nil, fmt.Errorf("HTTP is disabled offline: %s %s", req.Method, req.URL)
		}
		return t.fallback.RoundTrip(req)
	}
	L := luahttp.RequestState(req)
	if L == nil {
		return nil, fmt.Errorf("HTTP handler requires the invoking Lua thread")
	}
	headers := make(map[string]string, len(req.Header))
	for name := range req.Header {
		headers[name] = req.Header.Get(name)
	}
	request, err := codec.Marshal(L, struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}{req.Method, req.URL.String(), headers})
	if err != nil {
		return nil, err
	}
	top := L.GetTop()
	defer L.SetTop(top)
	if err := L.CallByParam(lua.P{Fn: t.handler, NRet: 2, Protect: true}, request); err != nil {
		// Assertions in a handler are test failures. Explicit nil,error results
		// below instead model a transport failure the plugin can handle.
		L.RaiseError("HTTP handler: %s", err)
	}
	if failure := L.Get(-1); failure != lua.LNil {
		return nil, fmt.Errorf("HTTP handler: %s", failure.String())
	}
	response, ok := L.Get(-2).(*lua.LTable)
	if !ok {
		L.RaiseError("HTTP handler must return a response table or nil, error")
	}
	var data struct {
		StatusCode int               `json:"status_code"`
		Body       string            `json:"body"`
		Headers    map[string]string `json:"headers"`
	}
	if err := codec.Unmarshal(response, &data); err != nil {
		L.RaiseError("HTTP handler response: %s", err)
	}
	if data.StatusCode < 100 || data.StatusCode > 599 {
		L.RaiseError("HTTP handler response requires a valid status_code")
	}
	responseHeaders := make(http.Header, len(data.Headers))
	for name, value := range data.Headers {
		responseHeaders.Set(name, value)
	}
	body := data.Body
	if req.Method == http.MethodHead {
		body = ""
	}
	return &http.Response{
		StatusCode:    data.StatusCode,
		Status:        fmt.Sprintf("%d %s", data.StatusCode, http.StatusText(data.StatusCode)),
		Header:        responseHeaders,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(data.Body)),
		Request:       req,
	}, nil
}

func (t *developmentTransport) close() {
	if closer, ok := t.fallback.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}
