// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reader_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/mindpak"
	"github.com/mindersec/minder/pkg/mindpak/reader"
)

func TestBundle_GetMetadata(t *testing.T) {
	t.Parallel()
	bundle := loadBundle(t, testDataPath)
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
			Name:          "GetProfile returns error for incorrect namespace",
			ProfileName:   "acmecorp/branch-protection-github-profile",
			ExpectedError: "invalid namespace",
		},
		{
			Name:          "GetProfile returns error for malformed name",
			ProfileName:   "acmecorp/foo/bar/branch-protection-github-profile",
			ExpectedError: "malformed profile name",
		},
		{
			Name:        "GetProfile retrieves profile in bundle",
			ProfileName: "branch-protection-github-profile",
		},
		{
			Name:        "GetProfile retrieves profile in bundle (file suffix)",
			ProfileName: "branch-protection-github-profile.yaml",
		},
		{
			Name:        "GetProfile retrieves profile in bundle (namespaced)",
			ProfileName: "stacklok/branch-protection-github-profile",
		},
	}

	// immutable - can be shared across parallel runs
	bundle := loadBundle(t, testDataPath)
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
				require.Equal(t, expectedProfileName, profile.GetName())
			}
		})
	}
}

func TestBundle_ForEachRuleType(t *testing.T) {
	t.Parallel()
	results := []string{}
	bundle := loadBundle(t, testDataPath)
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
	bundle := loadBundle(t, testDataPath)
	err := bundle.ForEachRuleType(func(_ *minderv1.RuleType) error {
		return errors.New(errorMessage)
	})
	require.ErrorContains(t, err, errorMessage)
}

func TestBundle_ForEachDataSource(t *testing.T) {
	t.Parallel()
	results := []string{}
	bundle := loadBundle(t, testDataPath)
	err := bundle.ForEachDataSource(func(dataSource *minderv1.DataSource) error {
		results = append(results, dataSource.Name)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"osv"}, results)
}

func TestBundle_ForEachDataSource_NoDataSources(t *testing.T) {
	t.Parallel()
	results := []string{}
	bundle := loadBundle(t, noDataSourcesPath)
	err := bundle.ForEachDataSource(func(dataSource *minderv1.DataSource) error {
		results = append(results, dataSource.Name)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{}, results)
}

func TestBundle_ForEachDataSourceError(t *testing.T) {
	t.Parallel()
	errorMessage := "oh no"
	bundle := loadBundle(t, testDataPath)
	err := bundle.ForEachDataSource(func(_ *minderv1.DataSource) error {
		return errors.New(errorMessage)
	})
	require.ErrorContains(t, err, errorMessage)
}

func loadBundle(t *testing.T, path string) reader.BundleReader {
	t.Helper()
	bundle, err := mindpak.NewBundleFromDirectory(path)
	if err != nil {
		t.Fatalf("Unable to load test data from %s: %v", path, err)
	}
	return reader.NewBundleReader(bundle)
}

const (
	testDataPath        = "../testdata/t2"
	expectedProfileName = "branch-protection-github-profile"
	noDataSourcesPath   = "../testdata/no-data-sources"
)
