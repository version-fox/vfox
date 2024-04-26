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

package config

import "time"

// Cache is the cache configuration
// -1: never expire
// 0: never cache
type Cache struct {
	AvailableHookDuration time.Duration `yaml:"availableHook"` // Available hook result cache time
}

var (
	EmptyCache = &Cache{
		AvailableHookDuration: 12 * time.Hour,
	}
)