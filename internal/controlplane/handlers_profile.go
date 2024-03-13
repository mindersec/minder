// Copyright 2023 Stacklok, Inc
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

package controlplane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/logger"
	prof "github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/util"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// CreateProfile creates a profile for a project
func (s *Server) CreateProfile(ctx context.Context,
	cpr *minderv1.CreateProfileRequest) (*minderv1.CreateProfileResponse, error) {
	in := cpr.GetProfile()
	if err := in.Validate(); err != nil {
		if errors.Is(err, minderv1.ErrValidationFailed) {
			return nil, util.UserVisibleError(codes.InvalidArgument, "Couldn't create profile: %s", err)
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid profile: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// validate that project and provider are valid and exist in the db
	err := entityCtx.Validate(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// TODO: This will be removed once we decouple providers from profiles
	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, entityCtx.Project.ID)
	if err != nil {
		return nil, providerError(err)
	}

	newProfile, err := s.profiles.CreateProfile(ctx, entityCtx.Project.ID, &provider, in)
	if err != nil {
		// assumption: service layer is setting meaningful errors
		return nil, err
	}

	resp := &minderv1.CreateProfileResponse{
		Profile: newProfile,
	}

	return resp, nil
}

// DeleteProfile is a method to delete a profile
func (s *Server) DeleteProfile(ctx context.Context,
	in *minderv1.DeleteProfileRequest) (*minderv1.DeleteProfileResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)

	err := entityCtx.Validate(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	parsedProfileID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile ID")
	}

	profile, err := s.store.GetProfileByID(ctx, db.GetProfileByIDParams{
		ProjectID: entityCtx.Project.ID,
		ID:        parsedProfileID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "profile not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get profile: %s", err)
	}

	err = s.store.DeleteProfile(ctx, db.DeleteProfileParams{
		ID:        profile.ID,
		ProjectID: entityCtx.Project.ID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete profile: %s", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = profile.Provider
	logger.BusinessRecord(ctx).Project = profile.ProjectID
	logger.BusinessRecord(ctx).Profile = logger.Profile{Name: profile.Name, ID: profile.ID}

	return &minderv1.DeleteProfileResponse{}, nil
}

// ListProfiles is a method to get all profiles for a project
func (s *Server) ListProfiles(ctx context.Context,
	_ *minderv1.ListProfilesRequest) (*minderv1.ListProfilesResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)

	err := entityCtx.Validate(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	profiles, err := s.store.ListProfilesByProjectID(ctx, entityCtx.Project.ID)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get profiles: %s", err)
	}

	var resp minderv1.ListProfilesResponse
	resp.Profiles = make([]*minderv1.Profile, 0, len(profiles))
	for _, profile := range engine.MergeDatabaseListIntoProfiles(profiles) {
		resp.Profiles = append(resp.Profiles, profile)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = entityCtx.Provider.Name
	logger.BusinessRecord(ctx).Project = entityCtx.Project.ID

	return &resp, nil
}

// GetProfileById is a method to get a profile by id
func (s *Server) GetProfileById(ctx context.Context,
	in *minderv1.GetProfileByIdRequest) (*minderv1.GetProfileByIdResponse, error) {

	entityCtx := engine.EntityFromContext(ctx)

	err := entityCtx.Validate(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	parsedProfileID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile ID")
	}

	profile, err := getProfilePBFromDB(ctx, parsedProfileID, entityCtx, s.store)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "not found") {
			return nil, util.UserVisibleError(codes.NotFound, "profile not found")
		}

		return nil, status.Errorf(codes.Internal, "failed to get profile: %s", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = entityCtx.Provider.Name
	logger.BusinessRecord(ctx).Project = entityCtx.Project.ID
	logger.BusinessRecord(ctx).Profile = logger.Profile{Name: profile.Name, ID: parsedProfileID}

	return &minderv1.GetProfileByIdResponse{
		Profile: profile,
	}, nil
}

func getProfilePBFromDB(
	ctx context.Context,
	id uuid.UUID,
	entityCtx engine.EntityContext,
	querier db.ExtendQuerier,
) (*minderv1.Profile, error) {
	profiles, err := querier.GetProfileByProjectAndID(ctx, db.GetProfileByProjectAndIDParams{
		ProjectID: entityCtx.Project.ID,
		ID:        id,
	})
	if err != nil {
		return nil, err
	}

	pols := engine.MergeDatabaseGetIntoProfiles(profiles)
	if len(pols) == 0 {
		return nil, fmt.Errorf("profile not found")
	} else if len(pols) > 1 {
		return nil, fmt.Errorf("expected only one profile, got %d", len(pols))
	}

	// This should be only one profile
	for _, profile := range pols {
		return profile, nil
	}

	return nil, fmt.Errorf("profile not found")
}

func getRuleEvalEntityInfo(
	ctx context.Context,
	store db.Store,
	entityType *db.NullEntities,
	selector *uuid.NullUUID,
	rs db.ListRuleEvaluationsByProfileIdRow,
	providerName string,
) map[string]string {
	entityInfo := map[string]string{
		"provider": providerName,
	}

	if rs.RepositoryID.Valid {
		// this is always true now but might not be when we support entities not tied to a repo
		entityInfo["repo_name"] = rs.RepoName
		entityInfo["repo_owner"] = rs.RepoOwner
		entityInfo["repository_id"] = rs.RepositoryID.UUID.String()
	}

	if !selector.Valid || !entityType.Valid {
		return entityInfo
	}

	if entityType.Entities == db.EntitiesArtifact {
		artifact, err := store.GetArtifactByID(ctx, selector.UUID)
		if err != nil {
			log.Printf("error getting artifact: %v", err)
			return entityInfo
		}
		entityInfo["artifact_id"] = artifact.ID.String()
		entityInfo["artifact_name"] = artifact.ArtifactName
		entityInfo["artifact_type"] = artifact.ArtifactType
	}

	return entityInfo
}

// GetProfileStatusByName is a method to get profile status
// nolint:gocyclo // TODO: Refactor this to be more readable
func (s *Server) GetProfileStatusByName(ctx context.Context,
	in *minderv1.GetProfileStatusByNameRequest) (*minderv1.GetProfileStatusByNameResponse, error) {

	entityCtx := engine.EntityFromContext(ctx)

	err := entityCtx.Validate(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	dbProfileStatus, err := s.store.GetProfileStatusByNameAndProject(ctx, db.GetProfileStatusByNameAndProjectParams{
		ProjectID: entityCtx.Project.ID,
		Name:      in.Name,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "profile %q status not found", in.Name)
		}
		return nil, status.Errorf(codes.Unknown, "failed to get profile: %s", err)
	}

	var ruleEvaluationStatuses []*minderv1.RuleEvaluationStatus
	var selector *uuid.NullUUID
	var dbEntity *db.NullEntities
	var ruleType *sql.NullString
	var ruleName *sql.NullString

	if in.GetAll() {
		selector = &uuid.NullUUID{}
		dbEntity = &db.NullEntities{}
	} else if e := in.GetEntity(); e != nil {
		if !e.GetType().IsValid() {
			return nil, util.UserVisibleError(codes.InvalidArgument,
				"invalid entity type %s, please use one of %s",
				e.GetType(), entities.KnownTypesCSV())
		}
		selector = &uuid.NullUUID{}
		if err := selector.Scan(e.GetId()); err != nil {
			return nil, util.UserVisibleError(codes.InvalidArgument, "invalid entity ID in selector")
		}
		dbEntity = &db.NullEntities{Entities: entities.EntityTypeToDB(e.GetType()), Valid: true}
	}

	ruleType = &sql.NullString{
		String: in.GetRuleType(),
		Valid:  in.GetRuleType() != "",
	}

	// TODO: Remove deprecated 'rule' field from proto
	if !ruleType.Valid {
		//nolint:staticcheck // ignore SA1019: Deprecated field supported for backward compatibility
		ruleType = &sql.NullString{
			String: in.GetRule(),
			Valid:  in.GetRule() != "",
		}
	}

	ruleName = &sql.NullString{
		String: in.GetRuleName(),
		Valid:  in.GetRuleName() != "",
	}

	// TODO: Handle retrieving status for other types of entities
	if selector != nil {
		dbRuleEvaluationStatuses, err := s.store.ListRuleEvaluationsByProfileId(ctx, db.ListRuleEvaluationsByProfileIdParams{
			ProfileID:    dbProfileStatus.ID,
			EntityID:     *selector,
			EntityType:   *dbEntity,
			RuleTypeName: *ruleType,
			RuleName:     *ruleName,
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.Unknown, "failed to list rule evaluation status: %s", err)
		}

		ruleEvaluationStatuses = s.getRuleEvaluationStatuses(
			ctx, dbRuleEvaluationStatuses, dbProfileStatus.ID.String(),
			dbEntity, selector, entityCtx.Provider.Name,
		)
		// TODO: Add other entities once we have database entries for them
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = entityCtx.Provider.Name
	logger.BusinessRecord(ctx).Project = entityCtx.Project.ID
	logger.BusinessRecord(ctx).Profile = logger.Profile{Name: dbProfileStatus.Name, ID: dbProfileStatus.ID}

	return &minderv1.GetProfileStatusByNameResponse{
		ProfileStatus: &minderv1.ProfileStatus{
			ProfileId:     dbProfileStatus.ID.String(),
			ProfileName:   dbProfileStatus.Name,
			ProfileStatus: string(dbProfileStatus.ProfileStatus),
			LastUpdated:   timestamppb.New(dbProfileStatus.LastUpdated),
		},
		RuleEvaluationStatus: ruleEvaluationStatuses,
	}, nil
}

func (s *Server) getRuleEvaluationStatuses(
	ctx context.Context,
	dbRuleEvaluationStatuses []db.ListRuleEvaluationsByProfileIdRow,
	profileId string,
	dbEntity *db.NullEntities,
	selector *uuid.NullUUID,
	providerName string,
) []*minderv1.RuleEvaluationStatus {
	ruleEvaluationStatuses := make(
		[]*minderv1.RuleEvaluationStatus, 0, len(dbRuleEvaluationStatuses),
	)
	for _, dbRuleEvalStat := range dbRuleEvaluationStatuses {
		var guidance string

		// make sure all fields are valid
		if !dbRuleEvalStat.EvalStatus.Valid ||
			!dbRuleEvalStat.EvalDetails.Valid ||
			!dbRuleEvalStat.RemStatus.Valid ||
			!dbRuleEvalStat.RemDetails.Valid ||
			!dbRuleEvalStat.EvalLastUpdated.Valid {
			log.Print("error rule evaluation value not valid")
			continue
		}

		if dbRuleEvalStat.EvalStatus.EvalStatusTypes == db.EvalStatusTypesFailure ||
			dbRuleEvalStat.EvalStatus.EvalStatusTypes == db.EvalStatusTypesError {
			ruleTypeInfo, err := s.store.GetRuleTypeByID(ctx, dbRuleEvalStat.RuleTypeID)
			if err != nil {
				log.Printf("error getting rule type info: %v", err)
			} else {
				guidance = ruleTypeInfo.Guidance
			}
		}

		st := &minderv1.RuleEvaluationStatus{
			ProfileId:           profileId,
			RuleId:              dbRuleEvalStat.RuleTypeID.String(),
			RuleName:            dbRuleEvalStat.RuleTypeName,
			RuleTypeName:        dbRuleEvalStat.RuleTypeName,
			RuleDescriptionName: dbRuleEvalStat.RuleName,
			Entity:              string(dbRuleEvalStat.Entity),
			Status:              string(dbRuleEvalStat.EvalStatus.EvalStatusTypes),
			Details:             dbRuleEvalStat.EvalDetails.String,
			EntityInfo:          getRuleEvalEntityInfo(ctx, s.store, dbEntity, selector, dbRuleEvalStat, providerName),
			Guidance:            guidance,
			LastUpdated:         timestamppb.New(dbRuleEvalStat.EvalLastUpdated.Time),
			RemediationStatus:   string(dbRuleEvalStat.RemStatus.RemediationStatusTypes),
			RemediationDetails:  dbRuleEvalStat.RemDetails.String,
		}

		if dbRuleEvalStat.RemLastUpdated.Valid {
			st.RemediationLastUpdated = timestamppb.New(dbRuleEvalStat.RemLastUpdated.Time)
		}

		ruleEvaluationStatuses = append(ruleEvaluationStatuses, st)
	}
	return ruleEvaluationStatuses
}

// GetProfileStatusByProject is a method to get profile status for a project
func (s *Server) GetProfileStatusByProject(ctx context.Context,
	_ *minderv1.GetProfileStatusByProjectRequest) (*minderv1.GetProfileStatusByProjectResponse, error) {

	entityCtx := engine.EntityFromContext(ctx)

	err := entityCtx.Validate(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// read profile status
	dbstats, err := s.store.GetProfileStatusByProject(ctx, entityCtx.Project.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "profile statuses not found for project")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get profile status: %s", err)
	}

	res := &minderv1.GetProfileStatusByProjectResponse{
		ProfileStatus: make([]*minderv1.ProfileStatus, 0, len(dbstats)),
	}

	for _, dbstat := range dbstats {
		res.ProfileStatus = append(res.ProfileStatus, &minderv1.ProfileStatus{
			ProfileId:     dbstat.ID.String(),
			ProfileName:   dbstat.Name,
			ProfileStatus: string(dbstat.ProfileStatus),
		})
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = entityCtx.Provider.Name
	logger.BusinessRecord(ctx).Project = entityCtx.Project.ID

	return res, nil
}

// PatchProfile updates a profile for a project with a partial request
func (s *Server) PatchProfile(ctx context.Context, ppr *minderv1.PatchProfileRequest) (*minderv1.PatchProfileResponse, error) {
	patch := ppr.GetPatch()
	entityCtx := engine.EntityFromContext(ctx)

	err := entityCtx.Validate(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	if ppr.GetId() == "" {
		return nil, util.UserVisibleError(codes.InvalidArgument, "profile ID must be specified")
	}

	profileID, err := uuid.Parse(ppr.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "Malformed UUID")
	}

	params := db.UpdateProfileParams{ID: profileID, ProjectID: entityCtx.Project.ID}

	// we check the pointers explicitly because the zero value of a string is valid
	// value that means "use default" and we want to distinguish that from "not set in the patch"
	if patch.Remediate != nil {
		params.Remediate = validateActionType(patch.GetRemediate())
	}
	if patch.Alert != nil {
		params.Alert = validateActionType(patch.GetAlert())
	}

	// Update top-level profile db object
	_, err = s.store.UpdateProfile(ctx, params)
	if err != nil {
		return nil, util.UserVisibleError(codes.Internal, "error updating profile: %v", err)
	}

	updatedProfile, err := getProfilePBFromDB(ctx, profileID, entityCtx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get profile: %s", err)
	}

	resp := &minderv1.PatchProfileResponse{
		Profile: updatedProfile,
	}

	return resp, nil
}

// UpdateProfile updates a profile for a project
func (s *Server) UpdateProfile(ctx context.Context,
	cpr *minderv1.UpdateProfileRequest) (*minderv1.UpdateProfileResponse, error) {
	in := cpr.GetProfile()

	entityCtx := engine.EntityFromContext(ctx)

	err := entityCtx.Validate(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// TODO: This will be removed once we decouple providers from profiles
	provider, err := getProviderFromRequestOrDefault(ctx, s.store, in, entityCtx.Project.ID)
	if err != nil {
		return nil, providerError(err)
	}

	updatedProfile, err := s.profiles.UpdateProfile(ctx, entityCtx.Project.ID, &provider, in)
	if err != nil {
		// assumption: service layer sets sensible errors
		return nil, err
	}

	return &minderv1.UpdateProfileResponse{
		Profile: updatedProfile,
	}, nil
}

func getUnusedOldRuleStatuses(
	newRules, oldRules prof.RuleMapping,
) prof.RuleMapping {
	unusedRuleStatuses := make(prof.RuleMapping)

	for ruleTypeAndName, rule := range oldRules {
		if _, ok := newRules[ruleTypeAndName]; !ok {
			unusedRuleStatuses[ruleTypeAndName] = rule
		}
	}

	return unusedRuleStatuses
}

func getUnusedOldRuleTypes(newRules, oldRules prof.RuleMapping) []prof.EntityAndRuleTuple {
	var unusedRuleTypes []prof.EntityAndRuleTuple

	oldRulesTypeMap := make(map[string]prof.EntityAndRuleTuple)
	for ruleTypeAndName, rule := range oldRules {
		oldRulesTypeMap[ruleTypeAndName.RuleType] = rule
	}

	newRulesTypeMap := make(map[string]prof.EntityAndRuleTuple)
	for ruleTypeAndName, rule := range newRules {
		newRulesTypeMap[ruleTypeAndName.RuleType] = rule
	}

	for ruleType, rule := range oldRulesTypeMap {
		if _, ok := newRulesTypeMap[ruleType]; !ok {
			unusedRuleTypes = append(unusedRuleTypes, rule)
		}
	}

	return unusedRuleTypes
}

// validateActionType returns the appropriate remediate type or the
// NULL DB type if the input is invalid, thus letting the server run
// the profile with the default remediate type.
func validateActionType(r string) db.NullActionType {
	switch r {
	case "on":
		return db.NullActionType{ActionType: db.ActionTypeOn, Valid: true}
	case "off":
		return db.NullActionType{ActionType: db.ActionTypeOff, Valid: true}
	case "dry_run":
		return db.NullActionType{ActionType: db.ActionTypeDryRun, Valid: true}
	}

	return db.NullActionType{Valid: false}
}
