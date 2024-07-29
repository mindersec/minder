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

package github

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	internalpb "github.com/stacklok/minder/internal/proto"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// RepoToSelectorEntity converts a Repository to a SelectorEntity
func (_ *GitHub) RepoToSelectorEntity(_ context.Context, r *minderv1.Repository) *internalpb.SelectorEntity {
	fullName := fmt.Sprintf("%s/%s", r.GetOwner(), r.GetName())

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_REPOSITORIES,
		Name:       fullName,
		Entity: &internalpb.SelectorEntity_Repository{
			Repository: &internalpb.SelectorRepository{
				Name:      fullName,
				IsFork:    proto.Bool(r.GetIsFork()),
				IsPrivate: proto.Bool(r.GetIsPrivate()),
			},
		},
	}
}

// ArtifactToSelectorEntity converts an Artifact to a SelectorEntity
func (_ *GitHub) ArtifactToSelectorEntity(_ context.Context, a *minderv1.Artifact) *internalpb.SelectorEntity {
	fullName := fmt.Sprintf("%s/%s", a.GetOwner(), a.GetName())

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_ARTIFACTS,
		Name:       fullName,
		Entity: &internalpb.SelectorEntity_Artifact{
			Artifact: &internalpb.SelectorArtifact{
				Name: fullName,
				Type: a.GetType(),
			},
		},
	}
}

// PullRequestToSelectorEntity converts a Pull Request to a SelectorEntity
func (_ *GitHub) PullRequestToSelectorEntity(_ context.Context, pr *minderv1.PullRequest) *internalpb.SelectorEntity {
	fullName := fmt.Sprintf("%s/%s/%d", pr.GetRepoOwner(), pr.GetRepoName(), pr.GetNumber())

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_PULL_REQUESTS,
		Name:       fullName,
		Entity: &internalpb.SelectorEntity_PullRequest{
			PullRequest: &internalpb.SelectorPullRequest{
				Name: fullName,
			},
		},
	}
}
