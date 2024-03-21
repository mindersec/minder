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

package sources_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/marketplaces/bundles/sources"
)

// This test does not fit into the scenario structure below and is trivial
// Happy path is implicitly tested by the other tests
func TestNewSourceFromDirectory_Fails(t *testing.T) {
	t.Parallel()
	bundle, err := sources.NewSourceFromTarGZ(invalidPath)
	require.Nil(t, bundle)
	require.ErrorContains(t, err, "unable to load bundle")
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
			bundle, err := source.GetBundle(scenario.BundleNamespace, scenario.BundleName)
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
