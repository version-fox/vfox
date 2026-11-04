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

package sdk

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pterm/pterm"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/plugin"
)

func newDownloadTestSDK(t *testing.T) *impl {
	t.Helper()
	return &impl{
		Name:        "test-sdk",
		InstallPath: t.TempDir(),
		envContext: &env.RuntimeEnvContext{
			UserConfig: &config.Config{Proxy: config.EmptyProxy},
			PathMeta: &pathmeta.PathMeta{
				User: pathmeta.UserPaths{Home: t.TempDir()},
			},
		},
	}
}

func TestSdk_Download_ErrorStatusIsReported(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte("<html>the mirror refused the download</html>"))
			}))
			defer srv.Close()

			sdk := newDownloadTestSDK(t)
			u, err := url.Parse(srv.URL + "/repro.zip")
			if err != nil {
				t.Fatalf("parse url: %v", err)
			}

			_, err = sdk.Download(u, nil)
			if err == nil {
				t.Fatalf("Download() error = nil, want an error mentioning status %d", status)
			}
			want := fmt.Sprintf("%d %s", status, http.StatusText(status))
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("Download() error = %q, want it to mention %q", err, want)
			}

			entries, readErr := os.ReadDir(sdk.InstallPath)
			if readErr != nil && !os.IsNotExist(readErr) {
				t.Fatalf("read install dir: %v", readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("Download() saved %d file(s) for an HTTP %d response, want none", len(entries), status)
			}
		})
	}
}

func TestSdk_Download_NotFoundKeepsMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	sdk := newDownloadTestSDK(t)
	u, err := url.Parse(srv.URL + "/missing.zip")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	_, err = sdk.Download(u, nil)
	if err == nil || !strings.Contains(err.Error(), "source file not found") {
		t.Fatalf("Download() error = %v, want source file not found", err)
	}
}

func TestSdk_Download_SavesSuccessfulResponse(t *testing.T) {
	const body = "the archive bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	sdk := newDownloadTestSDK(t)
	u, err := url.Parse(srv.URL + "/repro.zip")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	path, err := sdk.Download(u, nil)
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

func TestSdk_Download_ErrorStatusRedactsPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>refused</html>"))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL + "/repro.zip")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	u.User = url.UserPassword("user", "s3cret")

	sdk := newDownloadTestSDK(t)
	_, err = sdk.Download(u, nil)
	if err == nil {
		t.Fatal("Download() error = nil, want an error")
	}
	msg := err.Error()
	if strings.Contains(msg, "s3cret") {
		t.Fatalf("Download() error = %q, want the password redacted", msg)
	}
	if !strings.Contains(msg, "user:xxxxx@") {
		t.Fatalf("Download() error = %q, want the redacted userinfo", msg)
	}
	if !strings.Contains(msg, "unexpected HTTP status 403 Forbidden") {
		t.Fatalf("Download() error = %q, want it to name the status", msg)
	}
}

func TestSdk_Download_RedactsRequestBuildError(t *testing.T) {
	// A space in the host is percent-encoded by URL.String, and http.NewRequest
	// rejects the encoded escape in the host.
	u := &url.URL{Scheme: "http", User: url.UserPassword("user", "s3cret"), Host: "in valid", Path: "/repro.zip", RawQuery: "token=abc123"}

	sdk := newDownloadTestSDK(t)
	_, err := sdk.Download(u, nil)
	if err == nil {
		t.Fatal("Download() error = nil, want a request build error")
	}
	if msg := err.Error(); strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
		t.Fatalf("Download() error = %q, want the credentials and the query token redacted", msg)
	}
}

func TestSdk_Download_RedactsConnectionError(t *testing.T) {
	u, err := url.Parse("http://user:s3cret@127.0.0.1:1/repro.zip?token=abc123")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	sdk := newDownloadTestSDK(t)
	_, err = sdk.Download(u, nil)
	if err == nil {
		t.Fatal("Download() error = nil, want a connection error")
	}
	msg := err.Error()
	if strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
		t.Fatalf("Download() error = %q, want the password and the query token redacted", msg)
	}
	if !strings.Contains(msg, "user:xxxxx@") || !strings.Contains(msg, "token=xxxxx") {
		t.Fatalf("Download() error = %q, want the redacted userinfo and query", msg)
	}
}

func TestSdk_MoveRemoteFileRedactsParseError(t *testing.T) {
	sdk := newDownloadTestSDK(t)
	err := sdk.moveRemoteFile(&plugin.PreInstallPackageItem{Path: "http://user:Zq9%ss@127.0.0.1/repro.zip"}, t.TempDir())
	if err == nil {
		t.Fatal("moveRemoteFile() error = nil, want a parse error")
	}
	if msg := err.Error(); strings.Contains(msg, "Zq9") || strings.Contains(msg, "%ss") {
		t.Fatalf("moveRemoteFile() error = %q, want the rejected URL credentials hidden", msg)
	}
}

func TestSdk_MoveLocalFileRedactsURL(t *testing.T) {
	sdk := newDownloadTestSDK(t)
	source := "ftp://user:s3cret@127.0.0.1:1/demo-1.0.0.zip?token=abc123"

	var out bytes.Buffer
	pterm.SetDefaultOutput(&out)
	defer pterm.SetDefaultOutput(os.Stdout)

	err := sdk.moveLocalFile(&plugin.PreInstallPackageItem{Path: source}, t.TempDir())
	if err == nil {
		t.Fatal("moveLocalFile() error = nil, want the missing source file reported")
	}
	if msg := err.Error(); strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
		t.Fatalf("moveLocalFile() error = %q, want the password and the query token redacted", msg)
	}
	printed := out.String()
	if !strings.Contains(printed, "Moving") {
		t.Fatalf("moveLocalFile() printed %q, want the progress line", printed)
	}
	if strings.Contains(printed, "s3cret") || strings.Contains(printed, "abc123") {
		t.Fatalf("moveLocalFile() printed %q, want the password and the query token redacted", printed)
	}
}
