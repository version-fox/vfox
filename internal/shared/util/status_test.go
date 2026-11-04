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

package util

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestUnexpectedStatusError(t *testing.T) {
	ok := &http.Response{StatusCode: http.StatusOK, Status: "200 OK"}
	if err := UnexpectedStatusError(ok, "http://user:s3cret@127.0.0.1/file.zip"); err != nil {
		t.Fatalf("UnexpectedStatusError(200) = %v, want nil", err)
	}

	resp := &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden"}
	err := UnexpectedStatusError(resp, "http://user:s3cret@127.0.0.1/file.zip?token=abc123")
	if err == nil {
		t.Fatalf("UnexpectedStatusError(403) = nil, want an error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "unexpected HTTP status 403 Forbidden") {
		t.Fatalf("error = %q, want it to name the status", msg)
	}
	if strings.Contains(msg, "s3cret") {
		t.Fatalf("error = %q, want the password redacted", msg)
	}
	if !strings.Contains(msg, "user:xxxxx@") {
		t.Fatalf("error = %q, want the redacted userinfo", msg)
	}
	if !strings.Contains(msg, "/file.zip") {
		t.Fatalf("error = %q, want the URL path to be kept", msg)
	}
}

func TestRedactSourceURL(t *testing.T) {
	for _, tt := range []struct {
		name string
		raw  string
		want string
	}{
		{"password", "http://user:s3cret@example.com/file.zip", "http://user:xxxxx@example.com/file.zip"},
		{"user name without password", "http://tok3n@example.com/file.zip", "http://****@example.com/file.zip"},
		{"query", "http://example.com/file.zip?token=abc123", "http://example.com/file.zip?token=xxxxx"},
		{"password with query", "http://user:s3cret@example.com/file.zip?token=abc123", "http://user:xxxxx@example.com/file.zip?token=xxxxx"},
		{"query part without equals", "http://example.com/file.zip?abc123", "http://example.com/file.zip?xxxxx"},
		{"query part without equals among keyed parts", "http://example.com/file.zip?abc123&b=2", "http://example.com/file.zip?xxxxx&b=xxxxx"},
		{"nothing to hide", "http://example.com/file.zip", "http://example.com/file.zip"},
		{"rejected by url.Parse", "http://user:Zq9%ss@example.com/file.zip", "http://xxxxx@example.com/file.zip"},
		{"rejected by url.Parse with a query", "http://example.com/%zz?token=abc123", "http://example.com/%zz"},
		{"rejected by url.Parse with a leading digit scheme", "1http://user:Zq9%ss@host/file.zip", "1http://user:Zq9%ss@host/file.zip"},
		{"absolute path", "/tmp/plugin.zip", "/tmp/plugin.zip"},
		{"relative path", "./plugin.zip", "./plugin.zip"},
		{"path with a bare percent", "/tmp/100%/plugin.zip", "/tmp/100%/plugin.zip"},
		{"local path containing a scheme separator", "/tmp/100%/dir://x@y", "/tmp/100%/dir://x@y"},
		{"windows path", `C:\plugins\plugin.zip`, `C:\plugins\plugin.zip`},
		{"windows path with slashes", "C:/plugins/plugin.zip", "C:/plugins/plugin.zip"},
		{"empty", "", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactSourceURL(tt.raw); got != tt.want {
				t.Errorf("RedactSourceURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRedactURLError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := RedactURLError(nil); got != nil {
			t.Fatalf("RedactURLError(nil) = %v, want nil", got)
		}
	})

	t.Run("connection error", func(t *testing.T) {
		err := &url.Error{
			Op:  "Get",
			URL: "http://user:s3cret@127.0.0.1:1/file.zip?token=abc123",
			Err: errors.New("dial tcp 127.0.0.1:1: connect: connection refused"),
		}
		redacted := RedactURLError(err)
		msg := redacted.Error()
		if strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
			t.Errorf("error = %q, want the password and the query token redacted", msg)
		}
		if !strings.Contains(msg, "user:xxxxx@") || !strings.Contains(msg, "token=xxxxx") {
			t.Errorf("error = %q, want the redacted userinfo and query", msg)
		}
		if !strings.Contains(msg, "connection refused") {
			t.Errorf("error = %q, want the reason kept", msg)
		}
		if err.URL == redacted.URL {
			t.Errorf("RedactURLError must not mutate the original error")
		}
	})

	t.Run("parse error with credentials", func(t *testing.T) {
		err := &url.Error{
			Op:  "parse",
			URL: "http://user:Zq9%ss@127.0.0.1/file.zip",
			Err: url.EscapeError("%ss"),
		}
		redacted := RedactURLError(err)
		msg := redacted.Error()
		if strings.Contains(msg, "Zq9") || strings.Contains(msg, "%ss") {
			t.Errorf("error = %q, want the rejected input hidden", msg)
		}
		if !strings.Contains(msg, `parse "http://xxxxx@127.0.0.1/file.zip"`) {
			t.Errorf("error = %q, want the redacted URL", msg)
		}
	})

	t.Run("parse error without secrets keeps the reason", func(t *testing.T) {
		err := &url.Error{
			Op:  "parse",
			URL: "http://127.0.0.1/%zz",
			Err: url.EscapeError("%zz"),
		}
		redacted := RedactURLError(err)
		if !strings.Contains(redacted.Error(), "%zz") {
			t.Errorf("error = %q, want the original reason kept", redacted.Error())
		}
	})
}

func TestRedactError(t *testing.T) {
	t.Run("url error", func(t *testing.T) {
		err := &url.Error{
			Op:  "Get",
			URL: "http://user:s3cret@127.0.0.1:1/file.zip?token=abc123",
			Err: errors.New("dial tcp 127.0.0.1:1: connect: connection refused"),
		}
		msg := RedactError(err).Error()
		if strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
			t.Errorf("error = %q, want the password and the query token redacted", msg)
		}
	})

	t.Run("wrapped url error", func(t *testing.T) {
		err := fmt.Errorf("download failed: %w", &url.Error{
			Op:  "Get",
			URL: "http://user:s3cret@127.0.0.1:1/file.zip",
			Err: errors.New("connection refused"),
		})
		msg := RedactError(err).Error()
		if strings.Contains(msg, "s3cret") {
			t.Errorf("error = %q, want the password redacted", msg)
		}
	})

	t.Run("other error unchanged", func(t *testing.T) {
		err := errors.New("plain failure")
		if got := RedactError(err); got != err {
			t.Errorf("RedactError() = %v, want the same error", got)
		}
	})

	t.Run("nil", func(t *testing.T) {
		if got := RedactError(nil); got != nil {
			t.Errorf("RedactError(nil) = %v, want nil", got)
		}
	})
}
