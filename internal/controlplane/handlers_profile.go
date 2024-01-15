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
	"github.com/stacklok/minder/internal/reconcilers"
	"github.com/stacklok/minder/internal/util"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// this is a tuple that allows us track rule instantiations
// and the entity they're associated with
type entityAndRuleTuple struct {
	Entity minderv1.Entity
	RuleID uuid.UUID
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

// CreateProfile creates a profile for a project
func (s *Server) CreateProfile(ctx context.Context,
	cpr *minderv1.CreateProfileRequest) (*minderv1.CreateProfileResponse, error) {
	in := cpr.GetProfile()

	ctx, err := s.contextValidation(ctx, cpr.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default project: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, entityCtx.GetProject().ID); err != nil {
		return nil, err
	}

	// If provider doesn't exist, return error
	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      entityCtx.GetProvider().Name,
		ProjectID: entityCtx.GetProject().ID})
	if err != nil {
		return nil, providerError(err)
	}

	if err := in.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid profile: %v", err)
	}

	rulesInProf, err := s.getAndValidateRulesFromProfile(ctx, in, entityCtx)
	if err != nil {
		var violation *engine.RuleValidationError
		if errors.As(err, &violation) {
			log.Printf("error validating rule: %v", violation)
			return nil, util.UserVisibleError(codes.InvalidArgument,
				"profile contained invalid rule '%s': %s", violation.RuleType, violation.Err)
		}

		log.Printf("error getting rule type: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}

	// Now that we know it's valid, let's persist it!
	tx, err := s.store.BeginTransaction()
	if err != nil {
		log.Printf("error starting transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}
	defer s.store.Rollback(tx)

	qtx := s.store.GetQuerierWithTransaction(tx)

	params := db.CreateProfileParams{
		Provider:  provider.Name,
		ProjectID: entityCtx.GetProject().ID,
		Name:      in.GetName(),
		Remediate: validateActionType(in.GetRemediate()),
		Alert:     validateActionType(in.GetAlert()),
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

	if err := tx.Commit(); err != nil {
		log.Printf("error committing transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating profile")
	}

	idStr := profile.ID.String()
	in.Id = &idStr
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

	return resp, nil
}

func createProfileRulesForEntity(
	ctx context.Context,
	entity minderv1.Entity,
	profile *db.Profile,
	qtx db.Querier,
	rules []*minderv1.Profile_Rule,
	rulesInProf map[string]entityAndRuleTuple,
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
		if err != nil {
			log.Printf("error creating rule instantiation: %v", err)
			return status.Errorf(codes.Internal, "error creating profile")
		}
	}

	return err
}

// DeleteProfile is a method to delete a profile
func (s *Server) DeleteProfile(ctx context.Context,
	in *minderv1.DeleteProfileRequest) (*minderv1.DeleteProfileResponse, error) {
	_, err := s.contextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default project: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, entityCtx.GetProject().ID); err != nil {
		return nil, err
	}

	parsedProfileID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile ID")
	}

	_, err = s.store.GetProfileByID(ctx, parsedProfileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "profile not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get profile: %s", err)
	}

	err = s.store.DeleteProfile(ctx, parsedProfileID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete profile: %s", err)
	}

	return &minderv1.DeleteProfileResponse{}, nil
}

// ListProfiles is a method to get all profiles for a project
func (s *Server) ListProfiles(ctx context.Context,
	in *minderv1.ListProfilesRequest) (*minderv1.ListProfilesResponse, error) {
	ctx, err := s.contextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default project: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, entityCtx.GetProject().ID); err != nil {
		return nil, err
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

	return &resp, nil
}

// GetProfileById is a method to get a profile by id
func (s *Server) GetProfileById(ctx context.Context,
	in *minderv1.GetProfileByIdRequest) (*minderv1.GetProfileByIdResponse, error) {
	ctx, err := s.contextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default project: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, entityCtx.GetProject().ID); err != nil {
		return nil, err
	}

	parsedProfileID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile ID")
	}

	prof, err := getProfilePBFromDB(ctx, parsedProfileID, entityCtx, s.store)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "not found") {
			return nil, util.UserVisibleError(codes.NotFound, "profile not found")
		}

		return nil, status.Errorf(codes.Internal, "failed to get profile: %s", err)
	}

	return &minderv1.GetProfileByIdResponse{
		Profile: prof,
	}, nil
}

func getProfilePBFromDB(
	ctx context.Context,
	id uuid.UUID,
	entityCtx *engine.EntityContext,
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
		return nil, fmt.Errorf("failed to get profile: %w", err)
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
	ctx, err := s.contextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default project: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, entityCtx.GetProject().ID); err != nil {
		return nil, err
	}

	dbstat, err := s.store.GetProfileStatusByNameAndProject(ctx, db.GetProfileStatusByNameAndProjectParams{
		ProjectID: entityCtx.Project.ID,
		Name:      in.Name,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "profile status not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get profile: %s", err)
	}

	var rulestats []*minderv1.RuleEvaluationStatus
	var selector *uuid.NullUUID
	var dbEntity *db.NullEntities
	var rule *sql.NullString

	if in.GetAll() {
		selector = &uuid.NullUUID{}
		dbEntity = &db.NullEntities{}
	} else if e := in.GetEntity(); e != nil {
		if !e.GetType().IsValid() {
			return nil, status.Errorf(codes.InvalidArgument,
				"invalid entity type %s, please use one of %s",
				e.GetType(), entities.KnownTypesCSV())
		}
		selector = &uuid.NullUUID{}
		if err := selector.Scan(e.GetId()); err != nil {
			return nil, util.UserVisibleError(codes.InvalidArgument, "invalid entity ID in selector")
		}
		dbEntity = &db.NullEntities{Entities: entities.EntityTypeToDB(e.GetType()), Valid: true}
	}

	if len(in.GetRule()) > 0 {
		rule = &sql.NullString{String: in.GetRule(), Valid: true}
	} else {
		rule = &sql.NullString{Valid: false}
	}

	// TODO: Handle retrieving status for other types of entities
	if selector != nil {
		dbrulestat, err := s.store.ListRuleEvaluationsByProfileId(ctx, db.ListRuleEvaluationsByProfileIdParams{
			ProfileID:  dbstat.ID,
			EntityID:   *selector,
			EntityType: *dbEntity,
			RuleName:   *rule,
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.Unknown, "failed to list rule evaluation status: %s", err)
		}

		rulestats = make([]*minderv1.RuleEvaluationStatus, 0, len(dbrulestat))
		for _, rs := range dbrulestat {
			rs := rs

			var guidance string

			// make sure all fields are valid
			if !rs.EvalStatus.Valid || !rs.EvalDetails.Valid || !rs.RemStatus.Valid || !rs.RemDetails.Valid || !rs.EvalLastUpdated.Valid {
				log.Print("error rule evaluation value not valid")
				continue
			}

			if rs.EvalStatus.EvalStatusTypes == db.EvalStatusTypesFailure || rs.EvalStatus.EvalStatusTypes == db.EvalStatusTypesError {
				ruleTypeInfo, err := s.store.GetRuleTypeByID(ctx, rs.RuleTypeID)
				if err != nil {
					log.Printf("error getting rule type info: %v", err)
				} else {
					guidance = ruleTypeInfo.Guidance
				}
			}

			st := &minderv1.RuleEvaluationStatus{
				ProfileId:          dbstat.ID.String(),
				RuleId:             rs.RuleTypeID.String(),
				RuleName:           rs.RuleTypeName,
				Entity:             string(rs.Entity),
				Status:             string(rs.EvalStatus.EvalStatusTypes),
				Details:            rs.EvalDetails.String,
				EntityInfo:         getRuleEvalEntityInfo(ctx, s.store, dbEntity, selector, rs, entityCtx.GetProvider().Name),
				Guidance:           guidance,
				LastUpdated:        timestamppb.New(rs.EvalLastUpdated.Time),
				RemediationStatus:  string(rs.RemStatus.RemediationStatusTypes),
				RemediationDetails: rs.RemDetails.String,
			}

			if rs.RemLastUpdated.Valid {
				st.RemediationLastUpdated = timestamppb.New(rs.RemLastUpdated.Time)
			}

			rulestats = append(rulestats, st)
		}

		// TODO: Add other entities once we have database entries for them
	}

	return &minderv1.GetProfileStatusByNameResponse{
		ProfileStatus: &minderv1.ProfileStatus{
			ProfileId:     dbstat.ID.String(),
			ProfileName:   dbstat.Name,
			ProfileStatus: string(dbstat.ProfileStatus),
			LastUpdated:   timestamppb.New(dbstat.LastUpdated),
		},
		RuleEvaluationStatus: rulestats,
	}, nil
}

// GetProfileStatusByProject is a method to get profile status for a project
func (s *Server) GetProfileStatusByProject(ctx context.Context,
	in *minderv1.GetProfileStatusByProjectRequest) (*minderv1.GetProfileStatusByProjectResponse, error) {
	ctx, err := s.contextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default project: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, entityCtx.GetProject().ID); err != nil {
		return nil, err
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

	return res, nil
}

// UpdateProfile updates a profile for a project
//
//nolint:gocyclo
func (s *Server) UpdateProfile(ctx context.Context,
	cpr *minderv1.UpdateProfileRequest) (*minderv1.UpdateProfileResponse, error) {
	in := cpr.GetProfile()

	ctx, err := s.contextValidation(ctx, cpr.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default project: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, entityCtx.GetProject().ID); err != nil {
		return nil, err
	}

	if err := in.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid profile: %v", err)
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

	rules, err := s.getAndValidateRulesFromProfile(ctx, in, entityCtx)
	if err != nil {
		var violation *engine.RuleValidationError
		if errors.As(err, &violation) {
			log.Printf("error validating rule: %v", violation)
			return nil, util.UserVisibleError(codes.InvalidArgument,
				"profile contained invalid rule '%s': %s", violation.RuleType, violation.Err)
		}

		log.Printf("error getting rule type: %v", err)
		return nil, status.Errorf(codes.Internal, "error updating profile")
	}

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
		if err := updateProfileRulesForEntity(ctx, ent, &profile, qtx, entRules, rules, oldRules); err != nil {
			return nil, err
		}
	}

	if err := deleteUnusedRulesFromProfile(ctx, &profile, oldRules, qtx); err != nil {
		return nil, status.Errorf(codes.Internal, "error updating profile: %v", err)
	}

	if err := deleteRuleStatusesForProfile(ctx, &profile, oldRules, qtx); err != nil {
		return nil, status.Errorf(codes.Internal, "error updating profile: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("error committing transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error updating profile")
	}

	idStr := profile.ID.String()
	in.Id = &idStr
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

	return resp, nil
}

func (s *Server) getAndValidateRulesFromProfile(
	ctx context.Context,
	prof *minderv1.Profile,
	entityCtx *engine.EntityContext,
) (map[string]entityAndRuleTuple, error) {
	// We capture the rule instantiations here so we can
	// track them in the db later.
	rulesInProf := map[string]entityAndRuleTuple{}

	err := engine.TraverseAllRulesForPipeline(prof, func(r *minderv1.Profile_Rule) error {
		// TODO: This will need to be updated to support
		// the hierarchy tree once that's settled in.
		rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
			Provider:  entityCtx.GetProvider().Name,
			ProjectID: entityCtx.GetProject().ID,
			Name:      r.GetType(),
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return &engine.RuleValidationError{
					Err:      fmt.Sprintf("cannot find rule type %s", r.GetType()),
					RuleType: r.GetType(),
				}
			}

			return fmt.Errorf("error getting rule type %s: %w", r.GetType(), err)
		}

		rtyppb, err := engine.RuleTypePBFromDB(&rtdb)
		if err != nil {
			return fmt.Errorf("cannot convert rule type %s to pb: %w", rtdb.Name, err)
		}

		rval, err := engine.NewRuleValidator(rtyppb)
		if err != nil {
			return fmt.Errorf("error creating rule validator: %w", err)
		}

		if err := rval.ValidateRuleDefAgainstSchema(r.Def.AsMap()); err != nil {
			return fmt.Errorf("error validating rule: %w", err)
		}

		if err := rval.ValidateParamsAgainstSchema(r.GetParams()); err != nil {
			return fmt.Errorf("error validating rule params: %w", err)
		}

		rulesInProf[r.GetType()] = entityAndRuleTuple{
			Entity: minderv1.EntityFromString(rtyppb.Def.InEntity),
			RuleID: rtdb.ID,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return rulesInProf, nil
}

func (s *Server) getRulesFromProfile(
	ctx context.Context,
	prof *minderv1.Profile,
	entityCtx *engine.EntityContext,
) (map[string]entityAndRuleTuple, error) {
	// We capture the rule instantiations here so we can
	// track them in the db later.
	rulesInProf := map[string]entityAndRuleTuple{}

	err := engine.TraverseAllRulesForPipeline(prof, func(r *minderv1.Profile_Rule) error {
		// TODO: This will need to be updated to support
		// the hierarchy tree once that's settled in.
		rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
			Provider:  entityCtx.GetProvider().Name,
			ProjectID: entityCtx.GetProject().ID,
			Name:      r.GetType(),
		})
		if err != nil {
			return fmt.Errorf("error getting rule type %s: %w", r.GetType(), err)
		}

		rtyppb, err := engine.RuleTypePBFromDB(&rtdb)
		if err != nil {
			return fmt.Errorf("cannot convert rule type %s to pb: %w", rtdb.Name, err)
		}

		rulesInProf[r.GetType()] = entityAndRuleTuple{
			Entity: minderv1.EntityFromString(rtyppb.Def.InEntity),
			RuleID: rtdb.ID,
		}

		return nil
	})

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
	rulesInProf map[string]entityAndRuleTuple,
	oldRulesInProf map[string]entityAndRuleTuple,
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

		// Remove the rule from the old rule IDs so we
		// can delete the ones that are no longer needed
		delete(oldRulesInProf, idx)
	}

	return err
}

func getProfileFromPBForUpdateWithQuerier(
	ctx context.Context,
	prof *minderv1.Profile,
	entityCtx *engine.EntityContext,
	querier db.ExtendQuerier,
) (*db.Profile, error) {
	if prof.GetId() != "" {
		return getProfileFromPBForUpdateByID(ctx, prof, querier)
	}

	return getProfileFromPBForUpdateByName(ctx, prof, entityCtx, querier)
}

func getProfileFromPBForUpdateByID(
	ctx context.Context,
	prof *minderv1.Profile,
	querier db.ExtendQuerier,
) (*db.Profile, error) {
	id, err := uuid.Parse(prof.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile ID")
	}

	pdb, err := querier.GetProfileByIDAndLock(ctx, id)
	if err != nil {
		return nil, err
	}

	return &pdb, nil
}

func getProfileFromPBForUpdateByName(
	ctx context.Context,
	prof *minderv1.Profile,
	entityCtx *engine.EntityContext,
	querier db.ExtendQuerier,
) (*db.Profile, error) {
	pdb, err := querier.GetProfileByNameAndLock(ctx, db.GetProfileByNameAndLockParams{
		Name:      prof.GetName(),
		ProjectID: entityCtx.GetProject().ID,
	})
	if err != nil {
		return nil, err
	}

	return &pdb, nil
}

func validateProfileUpdate(old *db.Profile, new *minderv1.Profile, entityCtx *engine.EntityContext) error {
	if old.Name != new.Name {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile name")
	}

	if old.ProjectID != entityCtx.Project.ID {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile project")
	}

	if old.Provider != entityCtx.GetProvider().Name {
		return util.UserVisibleError(codes.InvalidArgument, "cannot change profile provider")
	}

	return nil
}

func deleteUnusedRulesFromProfile(
	ctx context.Context,
	profile *db.Profile,
	unusedRules map[string]entityAndRuleTuple,
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
	unusedRules map[string]entityAndRuleTuple,
	querier db.ExtendQuerier,
) error {
	for _, rule := range unusedRules {
		log.Printf("deleting rule evaluations for rule %s in profile %s", rule.RuleID, profile.ID)
		if err := querier.DeleteRuleStatusesForProfileAndRuleType(ctx, db.DeleteRuleStatusesForProfileAndRuleTypeParams{
			ProfileID:  profile.ID,
			RuleTypeID: rule.RuleID,
		}); err != nil {
			log.Printf("error deleting rule evaluations: %v", err)
			return fmt.Errorf("error deleting rule evaluations: %w", err)
		}
	}

	return nil
}
