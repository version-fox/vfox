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

package commands

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
)

func TestDownloadFileRejectsUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>refused</html>"))
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "vfox.zip")
	err := downloadFile(srv.Client(), path, srv.URL+"/vfox.zip")
	if err == nil || !strings.Contains(err.Error(), "unexpected HTTP status 403 Forbidden") {
		t.Fatalf("downloadFile() error = %v, want unexpected HTTP status 403 Forbidden", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("downloadFile() created %q for an HTTP 403 response", path)
	}
}

func TestDownloadFileSavesSuccessfulResponse(t *testing.T) {
	const body = "binary bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "vfox.zip")
	if err := downloadFile(srv.Client(), path, srv.URL+"/vfox.zip"); err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(got) != body {
		t.Fatalf("downloaded content = %q, want %q", got, body)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// fetchLatestVersion names the status of a refused response instead of
// reporting that no version was found in the error page.
func TestFetchLatestVersionNamesTheStatus(t *testing.T) {
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Status:     "429 Too Many Requests",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("<html>rate limited</html>")),
		}, nil
	})}

	_, err := fetchLatestVersion(client, nil)
	if err == nil || !strings.Contains(err.Error(), "unexpected HTTP status 429 Too Many Requests") {
		t.Fatalf("fetchLatestVersion() error = %v, want unexpected HTTP status 429 Too Many Requests", err)
	}
}

func TestUpgradeWithGitHubMirror(t *testing.T) {
	for _, body := range []string{`{"tag_name":"v1.2.3"}`, "invalid", "{}"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/https://api.github.com/repos/version-fox/vfox/releases/latest" {
				http.NotFound(w, r)
				return
			}
			_, _ = io.WriteString(w, body)
		}))
		mirror := &config.GitHub{BaseURL: srv.URL + "/https://github.com/"}
		tag, err := fetchLatestVersion(srv.Client(), mirror)
		srv.Close()
		if body == "invalid" || body == "{}" {
			if err == nil {
				t.Fatalf("accepted release %s", body)
			}
			continue
		}
		if err != nil || tag != "v1.2.3" {
			t.Fatalf("tag = %q, err = %v", tag, err)
		}
		archive, diff := generateUrls("v1", tag, mirror)
		base := srv.URL + "/https://github.com/version-fox/vfox/"
		if archive != base+"releases/download/"+tag+"/"+constructBinaryName(tag) || diff != base+"compare/v1..."+tag {
			t.Fatalf("unexpected URLs %q, %q", archive, diff)
		}
	}
}

type endlessReleaseReader struct {
	prefix string
	read   int64
}

func (r *endlessReleaseReader) Read(p []byte) (int, error) {
	n := 0
	if int(r.read) < len(r.prefix) {
		n = copy(p, r.prefix[r.read:])
		r.read += int64(n)
	}
	for i := n; i < len(p); i++ {
		p[i] = 'a'
	}
	r.read += int64(len(p) - n)
	return len(p), nil
}

func TestFetchLatestVersionBoundsOversizedResponse(t *testing.T) {
	stream := &endlessReleaseReader{prefix: `{"tag_name":"v9.9.9","body":"`}
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(stream),
		}, nil
	})}

	mirror := &config.GitHub{BaseURL: "https://mirror.example.com"}
	if tag, err := fetchLatestVersion(client, mirror); err == nil {
		t.Fatalf("fetchLatestVersion() = %q, nil; want error for oversized response", tag)
	}
	if stream.read != maxReleaseResponseBytes {
		t.Fatalf("bytes read = %d, want %d", stream.read, maxReleaseResponseBytes)
	}
}
