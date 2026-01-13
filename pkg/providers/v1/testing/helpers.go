// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package testing provides common functions which can be used to implement provider tests.
package testing

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// CheckRegistrationExcept verifies that registration works for the accepted
// entity types, and fails for the unsupported ones, except for the specified
// types which have more complex registration methods.
func CheckRegistrationExcept(t *testing.T, provider provifv1.Provider, skip ...minderv1.Entity) {
	t.Helper()

	props := properties.NewProperties(map[string]any{
		"name":        "my-name",
		"upstream-id": "1234",
	})

	propsMap := make(map[string]*properties.Property, props.Len())
	for k, v := range props.Iterate() {
		propsMap[k] = v
	}

	cases := []minderv1.Entity{minderv1.Entity_ENTITY_UNSPECIFIED}
	// Unspecified is invalid, so start with the first value
	for i := minderv1.Entity_ENTITY_REPOSITORIES; i.IsValid(); i++ {
		if slices.Contains(skip, i) {
			continue
		}
		cases = append(cases, i)
	}

	for _, entType := range cases {
		t.Run(entType.ToString(), func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			newProps, err := provider.RegisterEntity(ctx, entType, props)

			if !provider.SupportsEntity(entType) {
				if !errors.Is(err, provifv1.ErrUnsupportedEntity) {
					t.Errorf("Expected unsupported entity for %s, got %q", entType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error for supported entity %s, need to skip? %q", entType, err)
			}

			newPropsMap := make(map[string]*properties.Property, newProps.Len())
			for k, v := range newProps.Iterate() {
				newPropsMap[k] = v
			}

			if diff := cmp.Diff(propsMap, newPropsMap); diff != "" {
				t.Errorf("Unexpected property update:\n%s", diff)
			}
		})
	}
}
