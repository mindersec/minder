//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cli contains utility for the cli
package cli

import (
	"testing"
)

func TestGetRepositoryName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		owner string
		name string
		want string
	}{
		{
			name: "test",
			want: "test",
		},
		{
			owner: "george",
			name: "test",
			want: "george/test",
		},
	}
	for _, tt := range tests {
		tc := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := GetRepositoryName(tc.owner, tc.name); got != tc.want {
				t.Errorf("GetRepositoryName() = %v, want %v", got, tc.want)
			}
		})
	}
}