// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

import (
	"testing"

	testhelper "github.com/mindersec/minder/pkg/providers/v1/testing"
)

func TestRegistration(t *testing.T) {
	t.Parallel()
	// We don't need a full constructor here, so we're naughty
	dh := &dockerHubImageLister{}
	testhelper.CheckRegistrationExcept(t, dh)
}
