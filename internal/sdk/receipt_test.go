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
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/plugin"
)

func TestReceiptRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := &Receipt{
		Schema:      receiptSchema,
		SDK:         "flutter",
		Version:     "3.47.4",
		Requested:   "stable",
		Origin:      OriginInstalled,
		InstalledAt: time.Date(2026, 10, 21, 10, 15, 3, 0, time.UTC),
		Vfox:        "1.0.13",
		Plugin:      ReceiptPlugin{Name: "flutter", Version: "0.2.5"},
		Sources:     []ReceiptSource{{Name: "flutter", Version: "3.47.4", URL: "https://example.com/flutter.tar.xz", Checksum: "sha256:abc"}},
		Facts:       map[string]string{"head": "9584c67"},
	}
	if err := writeReceipt(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := readReceipt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("receipt = %+v\nwant      %+v", got, want)
	}
}

func TestReadReceiptWithoutOne(t *testing.T) {
	// A version installed before receipts existed has none; that is not an error.
	receipt, err := readReceipt(t.TempDir())
	if receipt != nil || err != nil {
		t.Errorf("readReceipt = (%v, %v), want (nil, nil)", receipt, err)
	}
}

func TestReadCorruptReceipt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(receiptPath(dir), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	receipt, err := readReceipt(dir)
	if receipt != nil || err == nil {
		t.Errorf("readReceipt = (%v, %v), want an error", receipt, err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Error("a corrupt receipt is not a missing one")
	}
}

func TestReceiptIgnoresUnknownFields(t *testing.T) {
	dir := t.TempDir()
	body := `{"schema": 7, "sdk": "java", "version": "21", "origin": "linked", "target": "/jdk", "somethingNew": {"a": 1}}`
	if err := os.WriteFile(receiptPath(dir), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	receipt, err := readReceipt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Origin != OriginLinked || receipt.Target != "/jdk" || receipt.Schema != 7 {
		t.Errorf("receipt = %+v", receipt)
	}
}

func TestWriteReceiptReplacesAtomically(t *testing.T) {
	dir := t.TempDir()
	for _, head := range []string{"one", "two"} {
		if err := writeReceipt(dir, &Receipt{Schema: receiptSchema, Facts: map[string]string{"head": head}}); err != nil {
			t.Fatal(err)
		}
	}
	receipt, err := readReceipt(dir)
	if err != nil || receipt.Facts["head"] != "two" {
		t.Fatalf("receipt = %+v, err = %v", receipt, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != receiptFilename {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Errorf("the version directory holds %v, want only the receipt", names)
	}
}

func TestWriteReceiptFailsInAMissingDirectory(t *testing.T) {
	if err := writeReceipt(filepath.Join(t.TempDir(), "missing"), &Receipt{}); err == nil {
		t.Error("writing into a directory that does not exist must fail")
	}
}

func TestSanitizeSourceURL(t *testing.T) {
	tests := map[string]string{
		"https://user:secret@example.com/sdk.zip?token=abc&x=1#/name.zip": "https://example.com/sdk.zip",
		"https://example.com/a/b.tar.gz":                                  "https://example.com/a/b.tar.gz",
		"http://example.com:8080/sdk.zip?sig=1":                           "http://example.com:8080/sdk.zip",
		"/tmp/downloads/sdk.zip":                                          "/tmp/downloads/sdk.zip",
		`C:\Users\me\sdk.zip`:                                             `C:\Users\me\sdk.zip`,
		"":                                                                "",
	}
	for in, want := range tests {
		if got := sanitizeSourceURL(in); got != want {
			t.Errorf("sanitizeSourceURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReceiptSource(t *testing.T) {
	item := &plugin.PreInstallPackageItem{
		Name: "nodejs", Version: "24.21.0",
		Path:         "https://nodejs.org/dist/v24.21.0/node.tar.gz?auth=1",
		Headers:      map[string]string{"Authorization": "Bearer secret"},
		CheckSumItem: &plugin.CheckSumItem{Sha256: "deadbeef"},
	}
	source := receiptSource(item)
	want := ReceiptSource{Name: "nodejs", Version: "24.21.0", URL: "https://nodejs.org/dist/v24.21.0/node.tar.gz", Checksum: "sha256:deadbeef"}
	if source != want {
		t.Errorf("source = %+v, want %+v", source, want)
	}
	if strings.Contains(strings.ToLower(source.URL+source.Checksum), "secret") {
		t.Error("request headers must never reach the receipt")
	}

	if got := receiptSource(&plugin.PreInstallPackageItem{Name: "x", Version: "1"}); got.Checksum != "" {
		t.Errorf("no checksum was given, recorded %q", got.Checksum)
	}
}
