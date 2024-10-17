// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"fmt"
	"net/url"

	"github.com/rs/zerolog"
	"github.com/xanzy/go-gitlab"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func (c *gitlabClient) ListAllRepositories(ctx context.Context) ([]*minderv1.Repository, error) {
	groups := []*gitlab.Group{}
	if err := glRESTGet(ctx, c, "groups", &groups); err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	if len(groups) == 0 {
		zerolog.Ctx(ctx).Debug().Msg("no groups found")
		return nil, nil
	}

	var repos []*minderv1.Repository
	for _, g := range groups {
		projs := []*gitlab.Project{}
		path, err := url.JoinPath("groups", fmt.Sprintf("%d", g.ID), "projects")
		if err != nil {
			return nil, fmt.Errorf("failed to join URL path for projects: %w", err)
		}
		if err := glRESTGet(ctx, c, path, &projs); err != nil {
			return nil, fmt.Errorf("failed to get projects for group %s: %w", g.FullPath, err)
		}

		if repos == nil {
			repos = make([]*minderv1.Repository, 0, len(projs))
		}

		if len(projs) == 0 {
			zerolog.Ctx(ctx).Debug().Msgf("no projects found for group %s", g.FullPath)
		}

		for _, p := range projs {
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
	}

	zerolog.Ctx(ctx).Debug().Int("num_repos", len(repos)).Msg("found repositories in gitlab provider")

	return repos, nil
}
