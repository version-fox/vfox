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
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloaderRejectsUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>refused</html>"))
	}))
	defer srv.Close()

	localPath := t.TempDir()
	d := NewDownloader(localPath)
	u, err := url.Parse(srv.URL + "/repro.zip")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	if _, err = d.Download(u); err == nil || !strings.Contains(err.Error(), "unexpected HTTP status 403 Forbidden") {
		t.Fatalf("Download() error = %v, want unexpected HTTP status 403 Forbidden", err)
	}

	entries, readErr := os.ReadDir(localPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatalf("read local path: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("Download() saved %d file(s) for an HTTP 403 response, want none", len(entries))
	}
}

func TestDownloaderSavesSuccessfulResponse(t *testing.T) {
	const body = "archive bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	localPath := t.TempDir()
	d := NewDownloader(localPath)
	u, err := url.Parse(srv.URL + "/repro.zip")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	path, err := d.Download(u)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if filepath.Base(path) != "repro.zip" {
		t.Fatalf("Download() path = %q, want repro.zip", path)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(got) != body {
		t.Fatalf("downloaded content = %q, want %q", got, body)
	}
}

// The errors returned for a failed request must not echo the credentials or
// the query tokens of the URL, whether the request fails to parse or to
// connect.
func TestDownloaderRedactsRequestErrors(t *testing.T) {
	d := NewDownloader(t.TempDir())

	// A space in the host is percent-encoded by URL.String, and http.NewRequest
	// rejects the encoded escape in the host.
	unparsable := &url.URL{Scheme: "http", User: url.UserPassword("user", "s3cret"), Host: "in valid", Path: "/repro.zip", RawQuery: "token=abc123"}
	if _, err := d.Download(unparsable); err == nil {
		t.Fatal("Download() error = nil, want a parse error")
	} else if msg := err.Error(); strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
		t.Fatalf("Download() error = %q, want the credentials redacted", msg)
	}

	refused := &url.URL{Scheme: "http", User: url.UserPassword("user", "s3cret"), Host: "127.0.0.1:1", Path: "/repro.zip", RawQuery: "token=abc123"}
	if _, err := d.Download(refused); err == nil {
		t.Fatal("Download() error = nil, want a connection error")
	} else if msg := err.Error(); strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
		t.Fatalf("Download() error = %q, want the credentials and the query token redacted", msg)
	}
}
