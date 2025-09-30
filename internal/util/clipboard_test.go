/*
 *    Copyright 2025 Han Li and contributors
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
	"testing"
)

func TestCopyToClipboard(t *testing.T) {
	testText := "vfox use java@23.0.1+11"
	
	// CopyToClipboard should not panic and should return nil or an error
	err := CopyToClipboard(testText)
	
	// In CI/CD environment, clipboard utilities might not be available
	// So we just verify it doesn't panic
	if err != nil {
		t.Logf("CopyToClipboard returned error (expected in non-interactive environment): %v", err)
	} else {
		t.Log("CopyToClipboard succeeded")
	}
}
