// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package querier provides tools to interact with the Minder database
package querier

import (
	"context"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles"
)

// ProfileHandlers interface provides functions to interact with profiles
type ProfileHandlers interface {
	CreateProfile(
		ctx context.Context,
		projectID uuid.UUID,
		subscriptionID uuid.UUID,
		profile *pb.Profile,
	) (*pb.Profile, error)
	UpdateProfile(
		ctx context.Context,
		projectID uuid.UUID,
		subscriptionID uuid.UUID,
		profile *pb.Profile,
	) (*pb.Profile, error)
	DeleteProfile(ctx context.Context, projectID uuid.UUID, profileID uuid.UUID) error
	ListProfilesInstantiatingRuleType(ctx context.Context, ruleTypeID uuid.UUID) ([]string, error)
	GetProfileByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (map[string]*pb.Profile, error)
	DeleteRuleInstanceOfProfileInProject(ctx context.Context, projectID, profileID, ruleTypeID uuid.UUID) error
}

// DeleteProfile deletes a profile
func (q *querierType) DeleteProfile(ctx context.Context, projectID uuid.UUID, profileID uuid.UUID) error {
	if q.querier == nil {
		return ErrQuerierMissing
	}
	return q.querier.DeleteProfile(ctx, db.DeleteProfileParams{
		ProjectID: projectID,
		ID:        profileID,
	})
}

// UpdateProfile updates a profile
func (q *querierType) UpdateProfile(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	profile *pb.Profile,
) (*pb.Profile, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	if q.profileSvc == nil {
		return nil, ErrProfileSvcMissing
	}
	return q.profileSvc.UpdateProfile(ctx, projectID, subscriptionID, profile, q.querier)
}

// CreateProfile creates a profile
func (q *querierType) CreateProfile(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	profile *pb.Profile,
) (*pb.Profile, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	if q.profileSvc == nil {
		return nil, ErrProfileSvcMissing
	}
	return q.profileSvc.CreateProfile(ctx, projectID, subscriptionID, profile, q.querier)
}

// ListProfilesInstantiatingRuleType returns a list of profiles instantiating a rule type
func (q *querierType) ListProfilesInstantiatingRuleType(ctx context.Context, ruleTypeID uuid.UUID) ([]string, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	return q.querier.ListProfilesInstantiatingRuleType(ctx, ruleTypeID)
}

// GetProfileByProjectAndName returns a profile by project ID and name
func (q *querierType) GetProfileByProjectAndName(
	ctx context.Context,
	projectID uuid.UUID,
	name string,
) (map[string]*pb.Profile, error) {
	if q.querier == nil {
		return nil, ErrQuerierMissing
	}
	ret, err := q.querier.GetProfileByProjectAndName(ctx, db.GetProfileByProjectAndNameParams{
		Name:      name,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, err
	}
	return profiles.MergeDatabaseGetByNameIntoProfiles(ret), nil
}

// DeleteRuleInstanceOfProfileInProject deletes a rule instance for a profile in a project
func (q *querierType) DeleteRuleInstanceOfProfileInProject(
	ctx context.Context,
	projectID, profileID, ruleTypeID uuid.UUID,
) error {
	if q.querier == nil {
		return ErrQuerierMissing
	}
	return q.querier.DeleteRuleInstanceOfProfileInProject(ctx, db.DeleteRuleInstanceOfProfileInProjectParams{
		ProjectID:  projectID,
		ProfileID:  profileID,
		RuleTypeID: ruleTypeID,
	})
}
