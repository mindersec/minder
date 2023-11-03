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
	"log"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/auth"
	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/engine"
	"github.com/stacklok/mediator/internal/entities"
	"github.com/stacklok/mediator/internal/reconcilers"
	"github.com/stacklok/mediator/internal/util"
	minderv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

// authAndContextValidation is a helper function to initialize entity context info and validate input
// It also sets up the needed information in the `in` entity context that's needed for the rest of the flow
// Note that this also does an authorization check.
func (s *Server) authAndContextValidation(ctx context.Context, inout *minderv1.Context) (context.Context, error) {
	if inout == nil {
		return ctx, fmt.Errorf("context cannot be nil")
	}

	if err := s.ensureDefaultProjectForContext(ctx, inout); err != nil {
		return ctx, err
	}

	entityCtx, err := engine.GetContextFromInput(ctx, inout, s.store)
	if err != nil {
		return ctx, fmt.Errorf("cannot get context from input: %v", err)
	}

	if err := verifyValidProject(ctx, entityCtx); err != nil {
		return ctx, err
	}

	return engine.WithEntityContext(ctx, entityCtx), nil
}

// ensureDefaultProjectForContext ensures a valid group is set in the context or sets the default group
// if the group is not set in the incoming entity context, it'll set it.
func (s *Server) ensureDefaultProjectForContext(ctx context.Context, inout *minderv1.Context) error {
	// Project is already set
	if inout.GetProject() != "" {
		return nil
	}

	gid, err := auth.GetDefaultProject(ctx)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot infer group id")
	}

	g, err := s.store.GetProjectByID(ctx, gid)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot infer group id")
	}

	inout.Project = &g.Name
	return nil
}

// verifyValidProject verifies that the group is valid and the user is authorized to access it
// TODO: This will have to change once we have the hierarchy tree in place.
func verifyValidProject(ctx context.Context, in *engine.EntityContext) error {
	if !auth.IsAuthorizedForProject(ctx, in.GetProject().GetID()) {
		return status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	return nil
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

// CreateProfile creates a profile for a group
// nolint: gocyclo
func (s *Server) CreateProfile(ctx context.Context,
	cpr *minderv1.CreateProfileRequest) (*minderv1.CreateProfileResponse, error) {
	in := cpr.GetProfile()

	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// If provider doesn't exist, return error
	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      entityCtx.GetProvider().Name,
		ProjectID: entityCtx.GetProject().ID})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	if err := in.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid profile: %v", err)
	}

	// We capture the rule instantiations here so we can
	// track them in the db later.
	ruleIDs := map[string]uuid.UUID{}

	err = engine.TraverseAllRulesForPipeline(in, func(r *minderv1.Profile_Rule) error {
		// TODO: This will need to be updated to support
		// the hierarchy tree once that's settled in.
		rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
			Provider:  provider.Name,
			ProjectID: entityCtx.GetProject().GetID(),
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

		rtyppb, err := engine.RuleTypePBFromDB(&rtdb, entityCtx)
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

		ruleIDs[r.GetType()] = rtdb.ID

		return nil
	})

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
		ProjectID: entityCtx.GetProject().GetID(),
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
		if err := createProfileRulesForEntity(ctx, ent, &profile, qtx, entRules, ruleIDs); err != nil {
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
	ruleIDs map[string]uuid.UUID,
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

	for idx := range ruleIDs {
		ruleID := ruleIDs[idx]

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
	_, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
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

// ListProfiles is a method to get all profiles for a group
func (s *Server) ListProfiles(ctx context.Context,
	in *minderv1.ListProfilesRequest) (*minderv1.ListProfilesResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	profiles, err := s.store.ListProfilesByProjectID(ctx, entityCtx.Project.ID)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get profiles: %s", err)
	}

	var resp minderv1.ListProfilesResponse
	resp.Profiles = make([]*minderv1.Profile, 0, len(profiles))
	for _, profile := range engine.MergeDatabaseListIntoProfiles(profiles, entityCtx) {
		resp.Profiles = append(resp.Profiles, profile)
	}

	return &resp, nil
}

// GetProfileById is a method to get a profile by id
func (s *Server) GetProfileById(ctx context.Context,
	in *minderv1.GetProfileByIdRequest) (*minderv1.GetProfileByIdResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	parsedProfileID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid profile ID")
	}

	profiles, err := s.store.GetProfileByProjectAndID(ctx, db.GetProfileByProjectAndIDParams{
		ProjectID: entityCtx.Project.ID,
		ID:        parsedProfileID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get profile: %s", err)
	}

	var resp minderv1.GetProfileByIdResponse
	pols := engine.MergeDatabaseGetIntoProfiles(profiles, entityCtx)
	if len(pols) == 0 {
		return nil, status.Errorf(codes.NotFound, "profile not found")
	} else if len(pols) > 1 {
		return nil, status.Errorf(codes.Unknown, "failed to get profile: %s", err)
	}

	// This should be only one profile
	for _, profile := range pols {
		resp.Profile = profile
	}

	return &resp, nil
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
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

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
				log.Println("error rule evaluation value not valid")
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

// GetProfileStatusByProject is a method to get profile status for a group
func (s *Server) GetProfileStatusByProject(ctx context.Context,
	in *minderv1.GetProfileStatusByProjectRequest) (*minderv1.GetProfileStatusByProjectResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// read profile status
	dbstats, err := s.store.GetProfileStatusByProject(ctx, entityCtx.Project.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "profile statuses not found for group")
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
