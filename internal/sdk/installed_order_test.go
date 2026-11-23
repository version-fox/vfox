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
	"slices"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

func TestInstalledListIsNewestFirst(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	capturePterm(t)
	for _, v := range []Version{"dev", "18.0.0", "v20.1.0", "2.0.0-rc1", "9.0.0", "v22.0.0", "2.0.0", "10.0.0", "latest", "2.0.0-rc2"} {
		mustInstall(t, s, v)
	}
	want := []Version{"v22.0.0", "v20.1.0", "18.0.0", "10.0.0", "9.0.0", "2.0.0", "2.0.0-rc2", "2.0.0-rc1", "latest", "dev"}
	if got := s.InstalledList(); !slices.Equal(got, want) {
		t.Errorf("InstalledList() = %v, want %v", got, want)
	}
}

func TestCompareInstalledTreatsMissingPartsAsZero(t *testing.T) {
	if c := compareInstalled("1.0", "1.0.0"); c != 0 {
		t.Errorf("compareInstalled(1.0, 1.0.0) = %d, want 0", c)
	}
}

func TestInstalledListComparesTheNumbersInASuffix(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	capturePterm(t)
	for _, v := range []Version{"2.0.0-rc9", "2.0.0", "2.0.0-rc10", "2.0.0-beta.2", "2.0.0-beta.11", "2.0.0-beta"} {
		mustInstall(t, s, v)
	}
	want := []Version{"2.0.0", "2.0.0-rc10", "2.0.0-rc9", "2.0.0-beta.11", "2.0.0-beta.2", "2.0.0-beta"}
	if got := s.InstalledList(); !slices.Equal(got, want) {
		t.Errorf("InstalledList() = %v, want %v", got, want)
	}
}

// A partial version stands for the newest installed version that extends it,
// in the same order as InstalledList.
func TestPreUsePicksTheNewestVersion(t *testing.T) {
	for _, tc := range []struct {
		name      string
		installed []Version
		want      Version
	}{
		{"pre-releases only", []Version{"2.0.0-rc9", "2.0.0-rc10"}, "2.0.0-rc10"},
		{"a release and its pre-releases", []Version{"2.0.0-rc9", "2.0.0", "2.0.0-rc10"}, "2.0.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestSDK(t, &testPlugin{})
			capturePterm(t)
			for _, v := range tc.installed {
				mustInstall(t, s, v)
			}
			if got, err := s.preUse("2", env.Global); err != nil || got != tc.want {
				t.Errorf("preUse(2) = %q, %v, want %q", got, err, tc.want)
			}
		})
	}
}
