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

package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	EmptyCache = &Cache{
		AvailableHookDuration: CacheDuration(12 * time.Hour),
	}
)

// Cache is the cache configuration
type Cache struct {
	AvailableHookDuration CacheDuration `yaml:"availableHookDuration"` // Available hook result cache time
}

// CacheDuration is a duration that represents the cache duration and some special values
// -1: never expire
// 0: never cache
type CacheDuration time.Duration

func (d CacheDuration) MarshalYAML() (interface{}, error) {
	switch d {
	case -1:
		return -1, nil
	case 0:
		return 0, nil
	default:
		return d.String(), nil
	}
}

func (d *CacheDuration) UnmarshalYAML(node *yaml.Node) error {
	var data any
	err := node.Decode(&data)
	if err != nil {
		return err
	}
	switch va := data.(type) {
	case int:
		*d = CacheDuration(va)
	case string:
		pd, err := time.ParseDuration(va)
		if err != nil {
			return err
		}
		*d = CacheDuration(pd)
	}
	return nil
}

func (d CacheDuration) String() string {
	switch d {
	case -1:
		return "-1"
	case 0:
		return "0"
	}

	var str string
	duration := time.Duration(d)
	if h := int(duration.Hours()); h > 0 {
		str = fmt.Sprintf("%dh", h)
	}
	if m := int(duration.Minutes()) % 60; m > 0 {
		str = fmt.Sprintf("%s%dm", str, m)
	}
	if s := int(duration.Seconds()) % 60; s > 0 {
		str = fmt.Sprintf("%s%ds", str, s)
	}

	return str
}
