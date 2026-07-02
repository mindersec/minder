// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"testing"
)

func TestTestDir(t *testing.T) {
	t.Parallel()
	r := NewRunner()
	r.TestDir(t, "testdata")
}
