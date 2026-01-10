// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"testing"

	"github.com/mindersec/minder/internal/providers/github/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	testhelper "github.com/mindersec/minder/pkg/providers/v1/testing"
)

func TestRegistration(t *testing.T) {
	// We don't need a full constructor here, so we're naughty
	gh := &GitHub{
		propertyFetchers: properties.NewPropertyFetcherFactory(),
	}
	// Repositories do a bunch of special registration, so skip them
	// in this test -- we test them in common_test.go.
	testhelper.CheckRegistrationExcept(t, gh, minderv1.Entity_ENTITY_REPOSITORIES)
}
