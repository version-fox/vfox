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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
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
