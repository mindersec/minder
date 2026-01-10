// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ghcr

import (
	"testing"

	testhelper "github.com/mindersec/minder/pkg/providers/v1/testing"
)

func TestRegistration(t *testing.T) {
	// We don't need a full constructor here, so we're naughty
	il := &ImageLister{}
	testhelper.CheckRegistrationExcept(t, il)
}
