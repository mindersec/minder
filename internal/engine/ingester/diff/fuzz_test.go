// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package diff

import (
	"testing"
)

func FuzzDiffParse(f *testing.F) {
	f.Fuzz(func(_ *testing.T, param string, parser int) {
		switch parser % 3 {
		case 0:
			//nolint:gosec // The fuzzer does not validate the return values
			requirementsParse(param)
		case 1:
			//nolint:gosec // The fuzzer does not validate the return values
			npmParse(param)
		case 2:
			//nolint:gosec // The fuzzer does not validate the return values
			goParse(param)
		}
	})
}
