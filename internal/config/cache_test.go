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
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/shared/cache"
	"gopkg.in/yaml.v3"
)

func TestCacheDuration_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		cd      cache.Duration
		want    interface{}
		wantErr bool
	}{
		{"Negative", cache.Duration(-1), -1, false},
		{"Zero", cache.Duration(0), 0, false},
		{"Positive", cache.Duration(time.Hour), "1h", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cd.MarshalYAML()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MarshalYAML() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheDuration_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		node    *yaml.Node
		want    cache.Duration
		wantErr bool
	}{
		{"Negative", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "-1"}, cache.Duration(-1), false},
		{"Zero", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "0"}, cache.Duration(0), false},
		{"Positive", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "1h"}, cache.Duration(time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := cache.Duration(0)
			if err := cd.UnmarshalYAML(tt.node); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
			if cd != tt.want {
				t.Errorf("UnmarshalYAML() got = %v, want %v", cd, tt.want)
			}
		})
	}
}
