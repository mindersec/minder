// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reader_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/stacklok/minder/pkg/mindpak"
	"github.com/stacklok/minder/pkg/mindpak/reader"
)

func TestBundle_GetMetadata(t *testing.T) {
	t.Parallel()
	bundle := loadBundle(t)
	metadata := bundle.GetMetadata()
	require.NotNil(t, metadata)
	require.Equal(t, "t2", metadata.Name)
	require.Equal(t, "stacklok", metadata.Namespace)
	require.Equal(t, "v0.0.1", metadata.Version)
}

// TODO: the profile and rule type functions need some extra test cases which
// need some deliberately-broken bundle structures

func TestBundle_GetProfile(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		ProfileName   string
		ExpectedError string
	}{
		{
			Name:          "GetProfile returns error when profile does not exist",
			ProfileName:   "non-existent-profile",
			ExpectedError: "profile does not exist in bundle",
		},
		{
			Name:        "GetProfile retrieves profile in bundle",
			ProfileName: "branch-protection-github-profile",
		},
	}

	// immutable - can be shared across parallel runs
	bundle := loadBundle(t)
	for i := range scenarios {
		scenario := scenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			profile, err := bundle.GetProfile(scenario.ProfileName)
			if scenario.ExpectedError != "" {
				require.Nil(t, profile)
				require.ErrorContains(t, err, scenario.ExpectedError)
			} else {
				require.NoError(t, err)
				require.NotNil(t, profile)
				require.Equal(t, scenario.ProfileName, profile.GetName())
			}
		})
	}
}

func TestBundle_ForEachRuleType(t *testing.T) {
	t.Parallel()
	results := []string{}
	bundle := loadBundle(t)
	err := bundle.ForEachRuleType(func(ruleType *minderv1.RuleType) error {
		results = append(results, ruleType.Name)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"branch_protection_enabled"}, results)
}

func TestBundle_ForEachRuleTypeError(t *testing.T) {
	t.Parallel()
	errorMessage := "oh no"
	bundle := loadBundle(t)
	err := bundle.ForEachRuleType(func(_ *minderv1.RuleType) error {
		return errors.New(errorMessage)
	})
	require.ErrorContains(t, err, errorMessage)
}

func loadBundle(t *testing.T) reader.BundleReader {
	t.Helper()
	bundle, err := mindpak.NewBundleFromDirectory(testDataPath)
	if err != nil {
		t.Fatalf("Unable to load test data from %s: %v", testDataPath, err)
	}
	return reader.NewBundleReader(bundle)
}

const (
	testDataPath = "../testdata/t2"
)
