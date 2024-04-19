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

// Package repositories contains logic relating to the repository entity type
package repositories

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/util/ptr"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// PBRepositoryFromDB converts a database repository to a protobuf
func PBRepositoryFromDB(dbrepo db.Repository) *minderv1.Repository {
	return &minderv1.Repository{
		Id: ptr.Ptr(dbrepo.ID.String()),
		Context: &minderv1.Context{
			Provider: &dbrepo.Provider,
			Project:  ptr.Ptr(dbrepo.ProjectID.String()),
		},
		Owner:         dbrepo.RepoOwner,
		Name:          dbrepo.RepoName,
		RepoId:        dbrepo.RepoID,
		IsPrivate:     dbrepo.IsPrivate,
		IsFork:        dbrepo.IsFork,
		HookUrl:       dbrepo.WebhookUrl,
		DeployUrl:     dbrepo.DeployUrl,
		CloneUrl:      dbrepo.CloneUrl,
		DefaultBranch: dbrepo.DefaultBranch.String,
		License:       dbrepo.License.String,
		CreatedAt:     timestamppb.New(dbrepo.CreatedAt),
		UpdatedAt:     timestamppb.New(dbrepo.UpdatedAt),
	}
}
