// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package sources_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/mindpak"
	"github.com/mindersec/minder/pkg/mindpak/sources"
)

// This test does not fit into the scenario structure below and is trivial
// Happy path is implicitly tested by the other tests
func TestNewSourceFromDirectory_Fails(t *testing.T) {
	t.Parallel()
	bundle, err := sources.NewSourceFromTarGZ(invalidPath)
	require.Nil(t, bundle)
	require.ErrorContains(t, err, "unable to load bundle")
}

func TestListBundles(t *testing.T) {
	t.Parallel()
	bundle, err := sources.NewSourceFromTarGZ(sampleDataPath)
	require.NoError(t, err)
	idList, err := bundle.ListBundles()
	require.NoError(t, err)
	require.Len(t, idList, 1)
	require.Equal(t, idList[0].Name, "t2")
	require.Equal(t, idList[0].Namespace, "stacklok")
}

func TestSingleBundleSource_LoadBundle(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		Name            string
		BundleName      string
		BundleNamespace string
		ExpectedError   string
	}{
		{
			Name:            "LoadBundle returns error for non existent bundle",
			BundleName:      "foobar",
			BundleNamespace: "acmecorp",
			ExpectedError:   "bundle not found",
		},
		{
			Name:            "LoadBundle loads bundle",
			BundleName:      "t2",
			BundleNamespace: "stacklok",
		},
	}

	for i := range scenarios {
		scenario := scenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			source, err := sources.NewSourceFromTarGZ(sampleDataPath)
			require.NoError(t, err)
			id := mindpak.ID(scenario.BundleNamespace, scenario.BundleName)
			bundle, err := source.GetBundle(id)
			if scenario.ExpectedError == "" {
				require.Nil(t, err)
				require.Equal(t, scenario.BundleName, bundle.GetMetadata().Name)
				require.Equal(t, scenario.BundleNamespace, bundle.GetMetadata().Namespace)
			} else {
				require.Nil(t, bundle)
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

const (
	invalidPath    = "this is not a path"
	sampleDataPath = "testdata/bundle.tar.gz"
)
