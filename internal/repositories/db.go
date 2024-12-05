// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package repositories contains logic relating to the repository entity type
package repositories

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/util/ptr"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
