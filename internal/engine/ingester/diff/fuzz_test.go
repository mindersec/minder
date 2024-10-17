// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
