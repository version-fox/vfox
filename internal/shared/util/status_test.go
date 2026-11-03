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
