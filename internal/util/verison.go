/*
 *    Copyright 2024 Han Li and contributors
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
	"strconv"
	"strings"
)

func CompareVersion(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	len1 := len(parts1)
	len2 := len(parts2)

	// Get the maximum length between two versions
	maxLen := len1
	if len2 > maxLen {
		maxLen = len2
	}

	for i := 0; i < maxLen; i++ {
		// Because the length of v1 or v2 may be less than maxLen
		// We assume the missing part as 0
		part1 := 0
		if i < len1 {
			part1, _ = strconv.Atoi(parts1[i])
		}

		part2 := 0
		if i < len2 {
			part2, _ = strconv.Atoi(parts2[i])
		}

		if part1 != part2 {
			if part1 > part2 {
				return 1
			} else {
				return -1
			}
		}
	}

	// If all parts are equal
	return 0
}
