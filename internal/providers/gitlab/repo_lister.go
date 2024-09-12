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

package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/zerolog"
	"github.com/xanzy/go-gitlab"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func (c *gitlabClient) ListAllRepositories(ctx context.Context) ([]*minderv1.Repository, error) {
	groups := []*gitlab.Group{}
	if err := glREST(ctx, c, http.MethodGet, "groups", nil, &groups); err != nil {
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
		if err := glREST(ctx, c, http.MethodGet, path, nil, &projs); err != nil {
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
