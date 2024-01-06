/*
 *    Copyright 2024 [lihan aooohan@gmail.com]
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

import "testing"

func TestCompareVersion(t *testing.T) {
	type args struct {
		v1 string
		v2 string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "v1 > v2",
			args: args{
				v1: "1.0.0",
				v2: "0.0.1",
			},
			want: 1,
		},
		{
			name: "v1 > v2",
			args: args{
				v1: "0.2.3",
				v2: "0.2.1",
			},
			want: 1,
		},
		{
			name: "v1 > v2",
			args: args{
				v1: "3.2.0",
				v2: "1.2.1",
			},
			want: 1,
		},
		{
			name: "v1 < v2",
			args: args{
				v1: "0.0.1",
				v2: "1.0.0",
			},
			want: -1,
		},
		{
			name: "v1 < v2",
			args: args{
				v1: "0.1.1",
				v2: "0.2.0",
			},
			want: -1,
		},
		{
			name: "v1 = v2",
			args: args{
				v1: "1.0.0",
				v2: "1.0.0",
			},
			want: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CompareVersion(test.args.v1, test.args.v2); got != test.want {
				t.Errorf("CompareVersion() = %v, want %v", got, test.want)
			}
		})
	}
}
