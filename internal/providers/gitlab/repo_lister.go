// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// Access levels, from https://docs.gitlab.com/api/projects/#list-all-projects:
// 20 => "Reporter"
// 25 => "Security Manager"
// 30 => "Developer"
// 40 => "Maintainer"
// 50 => "Owner"
var minAccessLevelControl = 25 // "Security Manager"

func (c *gitlabClient) ListAllRepositories(ctx context.Context) ([]*minderv1.Repository, error) {
	managedProjects := []*gitlab.Project{}
	if err := glRESTGet(ctx, c, "projects?min_access_level=40", &managedProjects); err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	repos := make([]*minderv1.Repository, 0, len(managedProjects))
	for _, p := range managedProjects {
		props, err := gitlabProjectToProperties(p)
		if err != nil {
			return nil, fmt.Errorf("failed to convert project to properties: %w", err)
		}

		outRep, err := repoV1FromProperties(props)
		if err != nil {
			return nil, fmt.Errorf("failed to convert properties to repository: %w", err)
		}

		repos = append(repos, outRep)
	}

	zerolog.Ctx(ctx).Debug().Int("num_repos", len(repos)).Msg("found repositories in gitlab provider")

	return repos, nil
}
