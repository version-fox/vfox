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
)

// UnexpectedStatusError returns nil for HTTP 200 and otherwise an error naming
// the status and the source URL. Userinfo passwords in the URL are redacted, so
// a plugin URL with credentials cannot leak them into logs or terminal output.
// Callers that answer 404 with their own message should check for it first.
func UnexpectedStatusError(resp *http.Response, sourceURL string) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("unexpected HTTP status %s for %s", resp.Status, RedactSourceURL(sourceURL))
}

// RedactSourceURL masks the credentials and the query values of a URL, so a
// source URL can be shown in errors, messages and debug logs without leaking
// secrets. A user name that comes with a password stays visible with the
// password hidden, a user name on its own is masked the way pip masks it
// (****@host), and query keys are kept with their values masked. Anything that
// is not an absolute URL with a host, such as a local file path, is returned
// unchanged. A string that starts with scheme:// but that url.Parse rejects,
// for example a password with a bare %, loses everything between "://" and its
// last "@", and its query.
func RedactSourceURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return redactUnparsableURL(raw)
	}
	if u.Scheme == "" || u.Host == "" {
		return raw
	}
	if u.RawQuery != "" {
		u.RawQuery = redactQueryValues(u.RawQuery)
	}
	if u.User == nil {
		return u.String()
	}
	// The masked userinfo is rendered by hand, because url.Userinfo.String
	// escapes the "*" the masks use. A user name that comes with a password
	// stays visible; a user name on its own is masked the way pip masks it.
	var userinfo string
	if _, ok := u.User.Password(); ok {
		userinfo = url.User(u.User.Username()).String() + ":xxxxx@"
	} else {
		userinfo = "****@"
	}
	u.User = nil
	s := u.String()
	// u has a scheme and a host here, and Parse lowercases the scheme, so the
	// rendered URL always starts with scheme + "://".
	n := len(u.Scheme) + len("://")
	return s[:n] + userinfo + s[n:]
}

// redactQueryValues keeps every key of a query string and masks its value, so
// a token passed as ?token=... cannot leak while the parameter names stay
// readable. A part without "=" carries no key and is masked as a whole.
func redactQueryValues(rawQuery string) string {
	parts := strings.Split(rawQuery, "&")
	for i, part := range parts {
		if key, _, found := strings.Cut(part, "="); found {
			parts[i] = key + "=xxxxx"
		} else {
			parts[i] = "xxxxx"
		}
	}
	return strings.Join(parts, "&")
}

// redactUnparsableURL handles a string that starts with scheme:// but that
// url.Parse rejected, for example a password with a bare %: everything between
// "://" and the last "@" is hidden and the query is dropped, so the rejected
// credential is not echoed back as given.
func redactUnparsableURL(raw string) string {
	scheme, rest, found := strings.Cut(raw, "://")
	if !found || !isURLScheme(scheme) {
		return raw
	}
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		rest = "xxxxx@" + rest[i+1:]
	}
	if i := strings.IndexByte(rest, '?'); i >= 0 {
		rest = rest[:i]
	}
	return scheme + "://" + rest
}

// isURLScheme reports whether s follows the scheme syntax of RFC 3986, so a
// Windows path or a local path that merely contains "://" is left alone.
func isURLScheme(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z':
		case '0' <= c && c <= '9', c == '+', c == '-', c == '.':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return len(s) > 0
}

// RedactURLError returns a copy of err with its URL passed through
// RedactSourceURL, so the errors that net/http and net/url build cannot echo
// credentials or query tokens: net/http masks only the password, and net/url
// quotes the input it rejects, so a parse error whose URL had something to
// hide also gets a reason that does not quote the input.
func RedactURLError(err *url.Error) *url.Error {
	if err == nil {
		return nil
	}
	redacted := *err
	redacted.URL = RedactSourceURL(err.URL)
	if err.Op == "parse" && redacted.URL != err.URL {
		redacted.Err = errors.New("invalid URL")
	}
	return &redacted
}

// RedactError passes a *url.Error found in err through RedactURLError and
// returns any other error unchanged.
func RedactError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return RedactURLError(urlErr)
	}
	return err
}
