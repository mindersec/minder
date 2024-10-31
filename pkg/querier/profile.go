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
func (t *Type) DeleteProfile(ctx context.Context, projectID uuid.UUID, profileID uuid.UUID) error {
	return t.db.querier.DeleteProfile(ctx, db.DeleteProfileParams{
		ProjectID: projectID,
		ID:        profileID,
	})
}

// UpdateProfile updates a profile
func (t *Type) UpdateProfile(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	profile *pb.Profile,
) (*pb.Profile, error) {
	return t.profileSvc.UpdateProfile(ctx, projectID, subscriptionID, profile, t.db.querier)
}

// CreateProfile creates a profile
func (t *Type) CreateProfile(
	ctx context.Context,
	projectID uuid.UUID,
	subscriptionID uuid.UUID,
	profile *pb.Profile,
) (*pb.Profile, error) {
	return t.profileSvc.CreateProfile(ctx, projectID, subscriptionID, profile, t.db.querier)
}

// ListProfilesInstantiatingRuleType returns a list of profiles instantiating a rule type
func (t *Type) ListProfilesInstantiatingRuleType(ctx context.Context, ruleTypeID uuid.UUID) ([]string, error) {
	return t.db.querier.ListProfilesInstantiatingRuleType(ctx, ruleTypeID)
}

// GetProfileByProjectAndName returns a profile by project ID and name
func (t *Type) GetProfileByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (map[string]*pb.Profile, error) {
	ret, err := t.db.querier.GetProfileByProjectAndName(ctx, db.GetProfileByProjectAndNameParams{
		Name:      name,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, err
	}
	return profiles.MergeDatabaseGetByNameIntoProfiles(ret), nil
}

// DeleteRuleInstanceOfProfileInProject deletes a rule instance for a profile in a project
func (t *Type) DeleteRuleInstanceOfProfileInProject(ctx context.Context, projectID, profileID, ruleTypeID uuid.UUID) error {
	return t.db.querier.DeleteRuleInstanceOfProfileInProject(ctx, db.DeleteRuleInstanceOfProfileInProjectParams{
		ProjectID:  projectID,
		ProfileID:  profileID,
		RuleTypeID: ruleTypeID,
	})
}
