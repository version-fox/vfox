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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/version-fox/vfox/internal/plugin"
)

const (
	receiptFilename = ".vfox-receipt.json"
	receiptSchema   = 1
)

// Origin says who owns the files of an installed version.
type Origin string

const (
	// OriginInstalled: vfox downloaded and unpacked the version and may delete it.
	OriginInstalled Origin = "installed"
	// OriginLinked: the version lives elsewhere and belongs to someone else.
	// vfox only keeps a link to it and never modifies or deletes the target.
	OriginLinked Origin = "linked"
)

// ReceiptPlugin identifies the plugin that handled the installation.
type ReceiptPlugin struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ReceiptSource is a package the installation was made from.
type ReceiptSource struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	URL      string `json:"url,omitempty"`      // without credentials or query
	Checksum string `json:"checksum,omitempty"` // "<type>:<value>"
}

// Receipt is what vfox remembers about an installed version. It is written
// when the version is installed or linked and kept inside the version
// directory, so it disappears with it. A version installed before receipts
// existed has none; every check that needs one then stays silent.
type Receipt struct {
	Schema      int               `json:"schema"`
	SDK         string            `json:"sdk"`
	Version     string            `json:"version"`
	Requested   string            `json:"requested,omitempty"` // what the user asked for, e.g. "21" or "stable"
	Origin      Origin            `json:"origin"`
	Target      string            `json:"target,omitempty"` // linked: the directory the version points to
	InstalledAt time.Time         `json:"installedAt"`
	Vfox        string            `json:"vfox,omitempty"`
	Plugin      ReceiptPlugin     `json:"plugin"`
	Sources     []ReceiptSource   `json:"sources,omitempty"`
	Facts       map[string]string `json:"facts,omitempty"` // from the Fingerprint hook
}

func receiptPath(packagePath string) string {
	return filepath.Join(packagePath, receiptFilename)
}

// readReceipt returns (nil, nil) for a version that has no receipt.
func readReceipt(packagePath string) (*Receipt, error) {
	data, err := os.ReadFile(receiptPath(packagePath))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var receipt Receipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return nil, fmt.Errorf("parse %s: %w", receiptPath(packagePath), err)
	}
	return &receipt, nil
}

// writeReceipt replaces the receipt atomically, so a reader never sees half of one.
func writeReceipt(packagePath string, receipt *Receipt) error {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(packagePath, ".vfox-receipt-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	// Installations can be shared between users, so make the receipt readable.
	if err := os.Chmod(tmpName, 0644); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, receiptPath(packagePath)); err != nil {
		cleanup()
		return err
	}
	return nil
}

// sanitizeSourceURL drops what must not be stored: credentials, tokens in the
// query, and the fragment vfox uses as a file name hint. Local paths and
// anything that does not parse as a URL with a host are kept as they are.
func sanitizeSourceURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return raw
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func receiptSource(item *plugin.PreInstallPackageItem) ReceiptSource {
	source := ReceiptSource{Name: item.Name, Version: item.Version, URL: sanitizeSourceURL(item.Path)}
	if checksum := item.Checksum(); checksum != nil && checksum.Value != "" {
		source.Checksum = checksum.Type + ":" + checksum.Value
	}
	return source
}
