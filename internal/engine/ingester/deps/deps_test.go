// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package deps

import (
	"testing"

	"github.com/stretchr/testify/require"

	mock_github "github.com/mindersec/minder/internal/providers/github/mock"
	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestGetBranch(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		repo         *v1.Repository
		branch       string
		configBranch string
		expect       string
	}{
		{name: "default", expect: "main"},
		{name: "branch", branch: "test1", expect: "test1"},
		{name: "repo-default", repo: &v1.Repository{DefaultBranch: "defaultBranch"}, expect: "defaultBranch"},
		{name: "repo-default", configBranch: "ingestBranch", expect: "ingestBranch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gi, err := NewDepsIngester(&v1.DepsType{
				EntityType: &v1.DepsType_Repo{
					Repo: &v1.DepsType_RepoConfigs{
						Branch: tc.configBranch,
					},
				},
			}, &mock_github.MockGit{})
			require.NoError(t, err)

			branch := gi.getBranch(tc.repo, tc.branch)
			require.Equal(t, tc.expect, branch)
		})
	}
}
