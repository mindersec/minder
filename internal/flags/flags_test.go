//
// Copyright 2024 Stacklok, Inc.
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

// Package flags containts utilities for managing feature flags.
package flags

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/open-feature/go-sdk/openfeature"

	"github.com/stacklok/minder/internal/auth"
	config "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/engine/engcontext"
)

// nolint: tparallel
func TestOpenFeatureProviderFromFlags(t *testing.T) {
	t.Parallel()
	const testFlag = Experiment("test_flag")
	testFile := filepath.Clean(filepath.Join(t.TempDir(), "testfile.yaml"))
	tempFile, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() { _ = tempFile.Close() })
	configFile := `
test_flag:
  variations:
    FlagOn: true
    FlagOff: false
  defaultRule:
    variation: FlagOn
`
	if _, err := io.WriteString(tempFile, configFile); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	tests := []struct {
		name         string
		cfg          config.FlagsConfig
		expectedFlag bool
	}{{
		name: "No Config",
		cfg:  config.FlagsConfig{},
	}, {
		name: "No File",
		cfg: config.FlagsConfig{
			GoFeature: config.GoFeatureConfig{
				FilePath: "non-existent-file",
			},
		},
	}, {
		name: "File exists",
		cfg: config.FlagsConfig{
			GoFeature: config.GoFeatureConfig{
				FilePath: testFile,
			},
		},
		expectedFlag: true,
	}}
	//nolint: paralleltest
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These tests need to be exclusive with each other, because openfeature
			// uses a global variable to store the provider.
			// Other tests can mock the openfeature client to avoid this, but this test
			// specifically tests our interaction with the library, so we need exclusion here.

			ctx := context.Background()
			OpenFeatureProviderFromFlags(ctx, tt.cfg)

			client := openfeature.NewClient("test")
			userJWT := openid.New()
			if err := userJWT.Set("sub", "user-1"); err != nil {
				t.Fatalf("failed to set sub claim: %v", err)
			}
			ctx = auth.WithAuthTokenContext(ctx, userJWT)
			ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
				Project:  engcontext.Project{ID: uuid.New()},
				Provider: engcontext.Provider{Name: "testing"},
			})

			flagResult := Bool(ctx, client, testFlag)
			if flagResult != tt.expectedFlag {
				t.Errorf("expected %v, got %v", tt.expectedFlag, flagResult)
			}
		})
	}
}
