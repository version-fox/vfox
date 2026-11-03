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
	"fmt"
	"net/http"
	"net/url"
)

// UnexpectedStatusError returns nil for HTTP 200 and otherwise an error naming
// the status and the source URL. Userinfo passwords in the URL are redacted, so
// a plugin URL with credentials cannot leak them into logs or terminal output.
// Callers that answer 404 with their own message should check for it first.
func UnexpectedStatusError(resp *http.Response, sourceURL string) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("unexpected HTTP status %s for %s", resp.Status, redactSourceURL(sourceURL))
}

func redactSourceURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return raw
	}
	return u.Redacted()
}
