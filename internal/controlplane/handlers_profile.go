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
	"encoding/json"
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
	"github.com/stacklok/minder/internal/reconcilers"
	"github.com/stacklok/minder/internal/util"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

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

	rulesInProf, err := s.profileValidator.ValidateAndExtractRules(ctx, in, entityCtx)
	if err != nil {
		return nil, err
	}

	// Adds default rule names, if not present
	prof.PopulateRuleNames(in)

	// Now that we know it's valid, let's persist it!
	tx, err := s.store.BeginTransaction()
	if err != nil {
		log.Printf("error starting transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}
	defer s.store.Rollback(tx)

	qtx := s.store.GetQuerierWithTransaction(tx)

	params := db.CreateProfileParams{
		Provider:   entityCtx.Provider.Name,
		ProviderID: provider.ID,
		ProjectID:  entityCtx.Project.ID,
		Name:       in.GetName(),
		Remediate:  validateActionType(in.GetRemediate()),
		Alert:      validateActionType(in.GetAlert()),
	}

	// Create profile
	profile, err := qtx.CreateProfile(ctx, params)
	if db.ErrIsUniqueViolation(err) {
		log.Printf("profile already exists: %v", err)
		return nil, util.UserVisibleError(codes.AlreadyExists, "profile already exists")
	} else if err != nil {
		log.Printf("error creating profile: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}

	// Create entity rules entries
	for ent, entRules := range map[minderv1.Entity][]*minderv1.Profile_Rule{
		minderv1.Entity_ENTITY_REPOSITORIES:       in.GetRepository(),
		minderv1.Entity_ENTITY_ARTIFACTS:          in.GetArtifact(),
		minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS: in.GetBuildEnvironment(),
		minderv1.Entity_ENTITY_PULL_REQUESTS:      in.GetPullRequest(),
	} {
		if err := createProfileRulesForEntity(ctx, ent, &profile, qtx, entRules, rulesInProf); err != nil {
			return nil, err
		}
	}

	if err := s.store.Commit(tx); err != nil {
		log.Printf("error committing transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}

	idStr := profile.ID.String()
	in.Id = &idStr
	project := profile.ProjectID.String()
	in.Context = &minderv1.Context{
		Provider: &profile.Provider,
		Project:  &project,
	}
	resp := &minderv1.CreateProfileResponse{
		Profile: in,
	}

	msg, err := reconcilers.NewProfileInitMessage(entityCtx.Provider.Name, entityCtx.Project.ID)
	if err != nil {
		log.Printf("error creating reconciler event: %v", err)
		// error is non-fatal
		return resp, nil
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err := s.evt.Publish(reconcilers.InternalProfileInitEventTopic, msg); err != nil {
		log.Printf("error publishing reconciler event: %v", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = profile.Provider
	logger.BusinessRecord(ctx).Project = profile.ProjectID
	logger.BusinessRecord(ctx).Profile = logger.Profile{Name: profile.Name, ID: profile.ID}

	return resp, nil
}

func createProfileRulesForEntity(
	ctx context.Context,
	entity minderv1.Entity,
	profile *db.Profile,
	qtx db.Querier,
	rules []*minderv1.Profile_Rule,
	rulesInProf prof.RuleMapping,
) error {
	if rules == nil {
		return nil
	}

	marshalled, err := json.Marshal(rules)
	if err != nil {
		log.Printf("error marshalling %s rules: %v", entity, err)
		return status.Errorf(codes.Internal, "error creating profile")
	}
	entProf, err := qtx.CreateProfileForEntity(ctx, db.CreateProfileForEntityParams{
		ProfileID:       profile.ID,
		Entity:          entities.EntityTypeToDB(entity),
		ContextualRules: marshalled,
	})
	if err != nil {
		log.Printf("error creating profile for entity %s: %v", entity, err)
		return status.Errorf(codes.Internal, "error creating profile")
	}

	for idx := range rulesInProf {
		ruleRef := rulesInProf[idx]

		if ruleRef.Entity != entity {
			continue
		}

		ruleID := ruleRef.RuleID

		_, err := qtx.UpsertRuleInstantiation(ctx, db.UpsertRuleInstantiationParams{
			EntityProfileID: entProf.ID,
			RuleTypeID:      ruleID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("the rule instantiation for rule already existed.")
		} else if err != nil {
			log.Printf("error creating rule instantiation: %v", err)
			return status.Errorf(codes.Internal, "error creating profile")
		}
	}

	return err
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

	// The UpdateProfile accepts Null values for the remediate and alert fields as "the server default". Until
	// we fix that, we need to fetch the old profile to get the current values and extend the patch with those
	oldProfile, err := s.store.GetProfileByID(ctx, db.GetProfileByIDParams{
		ProjectID: entityCtx.Project.ID,
		ID:        profileID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "profile not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get profile: %s", err)
	}

	params := db.UpdateProfileParams{ID: profileID, ProjectID: entityCtx.Project.ID}

	// we check the pointers explicitly because the zero value of a string is valid
	// value that means "use default" and we want to distinguish that from "not set in the patch"
	if patch.Remediate != nil {
		params.Remediate = validateActionType(patch.GetRemediate())
	} else {
		params.Remediate = oldProfile.Remediate
	}

	if patch.Alert != nil {
		params.Alert = validateActionType(patch.GetAlert())
	} else {
		params.Alert = oldProfile.Alert
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
//
//nolint:gocyclo
func (s *Server) UpdateProfile(ctx context.Context,
	cpr *minderv1.UpdateProfileRequest) (*minderv1.UpdateProfileResponse, error) {
	in := cpr.GetProfile()

	entityCtx := engine.EntityFromContext(ctx)

	err := entityCtx.Validate(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// Previously, this validation was called later in this method. There does
	// not seem to be reason why it can't happen earlier, and it provides an
	// additional layer of validation before beginning the DB transaction.
	rules, err := s.profileValidator.ValidateAndExtractRules(ctx, in, entityCtx)
	if err != nil {
		return nil, err
	}

	tx, err := s.store.BeginTransaction()
	if err != nil {
		log.Printf("error starting transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error updating profile")
	}
	defer s.store.Rollback(tx)

	qtx := s.store.GetQuerierWithTransaction(tx)

	// Get object and ensure we lock it for update
	oldDBProfile, err := getProfileFromPBForUpdateWithQuerier(ctx, in, entityCtx, qtx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "profile not found")
		}

		return nil, status.Errorf(codes.Internal, "error fetching profile to be updated: %v", err)
	}

	// validate update
	if err := validateProfileUpdate(oldDBProfile, in, entityCtx); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile update: %v", err)
	}

	// Adds default rule names, if not present
	prof.PopulateRuleNames(in)

	oldProfile, err := getProfilePBFromDB(ctx, oldDBProfile.ID, entityCtx, qtx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "not found") {
			return nil, util.UserVisibleError(codes.NotFound, "profile not found")
		}

		return nil, status.Errorf(codes.Internal, "failed to get profile: %s", err)
	}

	oldRules, err := s.getRulesFromProfile(ctx, oldProfile, entityCtx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "profile not found")
		}
		return nil, status.Errorf(codes.Internal, "error fetching profile to be updated: %v", err)
	}

	// Update top-level profile db object
	profile, err := qtx.UpdateProfile(ctx, db.UpdateProfileParams{
		ProjectID: entityCtx.Project.ID,
		ID:        oldDBProfile.ID,
		Remediate: validateActionType(in.GetRemediate()),
		Alert:     validateActionType(in.GetAlert()),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error updating profile: %v", err)
	}

	// Create entity rules entries
	for ent, entRules := range map[minderv1.Entity][]*minderv1.Profile_Rule{
		minderv1.Entity_ENTITY_REPOSITORIES:       in.GetRepository(),
		minderv1.Entity_ENTITY_ARTIFACTS:          in.GetArtifact(),
		minderv1.Entity_ENTITY_BUILD_ENVIRONMENTS: in.GetBuildEnvironment(),
		minderv1.Entity_ENTITY_PULL_REQUESTS:      in.GetPullRequest(),
	} {
		if err := updateProfileRulesForEntity(ctx, ent, &profile, qtx, entRules, rules); err != nil {
			return nil, err
		}
	}

	unusedRuleStatuses := getUnusedOldRuleStatuses(rules, oldRules)
	unusedRuleTypes := getUnusedOldRuleTypes(rules, oldRules)

	if err := deleteUnusedRulesFromProfile(ctx, &profile, unusedRuleTypes, qtx); err != nil {
		return nil, status.Errorf(codes.Internal, "error updating profile: %v", err)
	}

	if err := deleteRuleStatusesForProfile(ctx, &profile, unusedRuleStatuses, qtx); err != nil {
		return nil, status.Errorf(codes.Internal, "error updating profile: %v", err)
	}

	if err := s.store.Commit(tx); err != nil {
		log.Printf("error committing transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error updating profile")
	}

	idStr := profile.ID.String()
	in.Id = &idStr
	project := profile.ProjectID.String()
	in.Context = &minderv1.Context{
		Provider: &profile.Provider,
		Project:  &project,
	}
	resp := &minderv1.UpdateProfileResponse{
		Profile: in,
	}

	// re-trigger profile evaluation
	msg, err := reconcilers.NewProfileInitMessage(entityCtx.Provider.Name, entityCtx.Project.ID)
	if err != nil {
		log.Printf("error creating reconciler event: %v", err)
		// error is non-fatal
		return resp, nil
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err := s.evt.Publish(reconcilers.InternalProfileInitEventTopic, msg); err != nil {
		log.Printf("error publishing reconciler event: %v", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Provider = profile.Provider
	logger.BusinessRecord(ctx).Project = profile.ProjectID
	logger.BusinessRecord(ctx).Profile = logger.Profile{Name: profile.Name, ID: profile.ID}

	return resp, nil
}

func (s *Server) getRulesFromProfile(
	ctx context.Context,
	profile *minderv1.Profile,
	entityCtx engine.EntityContext,
) (prof.RuleMapping, error) {
	// We capture the rule instantiations here so we can
	// track them in the db later.
	rulesInProf := make(prof.RuleMapping)

	err := engine.TraverseAllRulesForPipeline(profile, func(r *minderv1.Profile_Rule) error {
		// TODO: This will need to be updated to support
		// the hierarchy tree once that's settled in.
		rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
			Provider:  entityCtx.Provider.Name,
			ProjectID: entityCtx.Project.ID,
			Name:      r.GetType(),
		})
		if err != nil {
			return fmt.Errorf("error getting rule type %s: %w", r.GetType(), err)
		}

		rtyppb, err := engine.RuleTypePBFromDB(&rtdb)
		if err != nil {
			return fmt.Errorf("cannot convert rule type %s to pb: %w", rtdb.Name, err)
		}

		key := prof.RuleTypeAndNamePair{
			RuleType: r.GetType(),
			RuleName: prof.ComputeRuleName(r),
		}

		rulesInProf[key] = prof.EntityAndRuleTuple{
			Entity: minderv1.EntityFromString(rtyppb.Def.InEntity),
			RuleID: rtdb.ID,
		}

		return nil
	},
	)

	if err != nil {
		return nil, err
	}

	return rulesInProf, nil
}

func updateProfileRulesForEntity(
	ctx context.Context,
	entity minderv1.Entity,
	profile *db.Profile,
	qtx db.Querier,
	rules []*minderv1.Profile_Rule,
	rulesInProf prof.RuleMapping,
) error {
	if len(rules) == 0 {
		return qtx.DeleteProfileForEntity(ctx, db.DeleteProfileForEntityParams{
			ProfileID: profile.ID,
			Entity:    entities.EntityTypeToDB(entity),
		})
	}

	marshalled, err := json.Marshal(rules)
	if err != nil {
		log.Printf("error marshalling %s rules: %v", entity, err)
		return status.Errorf(codes.Internal, "error creating profile")
	}
	entProf, err := qtx.UpsertProfileForEntity(ctx, db.UpsertProfileForEntityParams{
		ProfileID:       profile.ID,
		Entity:          entities.EntityTypeToDB(entity),
		ContextualRules: marshalled,
	})
	if err != nil {
		log.Printf("error updating profile for entity %s: %v", entity, err)
		return err
	}

	for idx := range rulesInProf {
		ruleRef := rulesInProf[idx]

		if ruleRef.Entity != entity {
			continue
		}

		_, err := qtx.UpsertRuleInstantiation(ctx, db.UpsertRuleInstantiationParams{
			EntityProfileID: entProf.ID,
			RuleTypeID:      ruleRef.RuleID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("the rule instantiation for rule already existed.")
		} else if err != nil {
			log.Printf("error creating rule instantiation: %v", err)
			return status.Errorf(codes.Internal, "error updating profile")
		}
	}

	return err
}

func getProfileFromPBForUpdateWithQuerier(
	ctx context.Context,
	profile *minderv1.Profile,
	entityCtx engine.EntityContext,
	querier db.ExtendQuerier,
) (*db.Profile, error) {
	if profile.GetId() != "" {
		return getProfileFromPBForUpdateByID(ctx, profile, entityCtx, querier)
	}

	return getProfileFromPBForUpdateByName(ctx, profile, entityCtx, querier)
}

func getProfileFromPBForUpdateByID(
	ctx context.Context,
	profile *minderv1.Profile,
	entityCtx engine.EntityContext,
	querier db.ExtendQuerier,
) (*db.Profile, error) {
	id, err := uuid.Parse(profile.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile ID")
	}

	pdb, err := querier.GetProfileByIDAndLock(ctx, db.GetProfileByIDAndLockParams{
		ID:        id,
		ProjectID: entityCtx.Project.ID,
	})
	if err != nil {
		return nil, err
	}

	return &pdb, nil
}

func getProfileFromPBForUpdateByName(
	ctx context.Context,
	profile *minderv1.Profile,
	entityCtx engine.EntityContext,
	querier db.ExtendQuerier,
) (*db.Profile, error) {
	pdb, err := querier.GetProfileByNameAndLock(ctx, db.GetProfileByNameAndLockParams{
		Name:      profile.GetName(),
		ProjectID: entityCtx.Project.ID,
	})
	if err != nil {
		return nil, err
	}

	return &pdb, nil
}

func validateProfileUpdate(old *db.Profile, new *minderv1.Profile, entityCtx engine.EntityContext) error {
	if old.Name != new.Name {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile name")
	}

	if old.ProjectID != entityCtx.Project.ID {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile project")
	}

	if old.Provider != entityCtx.Provider.Name {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile provider")
	}

	return nil
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

func deleteUnusedRulesFromProfile(
	ctx context.Context,
	profile *db.Profile,
	unusedRules []prof.EntityAndRuleTuple,
	querier db.ExtendQuerier,
) error {
	for _, rule := range unusedRules {
		// get entity profile
		log.Printf("getting profile for entity %s", rule.Entity)
		entProf, err := querier.GetProfileForEntity(ctx, db.GetProfileForEntityParams{
			ProfileID: profile.ID,
			Entity:    entities.EntityTypeToDB(rule.Entity),
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("skipping rule deletion for entity %s, profile not found", rule.Entity)
				continue
			}
			log.Printf("error getting profile for entity %s: %v", rule.Entity, err)
			return fmt.Errorf("error getting profile for entity %s: %w", rule.Entity, err)
		}

		log.Printf("deleting rule instantiation for rule %s for entity profile %s", rule.RuleID, entProf.ID)
		if err := querier.DeleteRuleInstantiation(ctx, db.DeleteRuleInstantiationParams{
			EntityProfileID: entProf.ID,
			RuleTypeID:      rule.RuleID,
		}); err != nil {
			log.Printf("error deleting rule instantiation: %v", err)
			return fmt.Errorf("error deleting rule instantiation: %w", err)
		}
	}

	return nil
}

func deleteRuleStatusesForProfile(
	ctx context.Context,
	profile *db.Profile,
	unusedRuleStatuses prof.RuleMapping,
	querier db.ExtendQuerier,
) error {
	for ruleTypeAndName, rule := range unusedRuleStatuses {
		log.Printf("deleting rule evaluations for rule %s in profile %s", rule.RuleID, profile.ID)

		if err := querier.DeleteRuleStatusesForProfileAndRuleType(ctx, db.DeleteRuleStatusesForProfileAndRuleTypeParams{
			ProfileID:  profile.ID,
			RuleTypeID: rule.RuleID,
			RuleName:   ruleTypeAndName.RuleName,
		}); err != nil {
			log.Printf("error deleting rule evaluations: %v", err)
			return fmt.Errorf("error deleting rule evaluations: %w", err)
		}
	}

	return nil
}
