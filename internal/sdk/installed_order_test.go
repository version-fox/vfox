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
