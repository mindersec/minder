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
	mediatorv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// authAndContextValidation is a helper function to initialize entity context info and validate input
// It also sets up the needed information in the `in` entity context that's needed for the rest of the flow
// Note that this also does an authorization check.
func (s *Server) authAndContextValidation(ctx context.Context, inout *mediatorv1.Context) (context.Context, error) {
	if inout == nil {
		return ctx, fmt.Errorf("context cannot be nil")
	}

	if err := s.ensureDefaultGroupForContext(ctx, inout); err != nil {
		return ctx, err
	}

	entityCtx, err := engine.GetContextFromInput(ctx, inout, s.store)
	if err != nil {
		return ctx, fmt.Errorf("cannot get context from input: %v", err)
	}

	if err := verifyValidGroup(ctx, entityCtx); err != nil {
		return ctx, err
	}

	return engine.WithEntityContext(ctx, entityCtx), nil
}

// ensureDefaultGroupForContext ensures a valid group is set in the context or sets the default group
// if the group is not set in the incoming entity context, it'll set it.
func (s *Server) ensureDefaultGroupForContext(ctx context.Context, inout *mediatorv1.Context) error {
	// Group is already set
	if inout.GetGroup() != "" {
		return nil
	}

	gid, err := auth.GetDefaultGroup(ctx)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot infer group id")
	}

	g, err := s.store.GetGroupByID(ctx, gid)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot infer group id")
	}

	inout.Group = &g.Name
	return nil
}

// verifyValidGroup verifies that the group is valid and the user is authorized to access it
// TODO: This will have to change once we have the hierarchy tree in place.
func verifyValidGroup(ctx context.Context, in *engine.EntityContext) error {
	if !auth.IsAuthorizedForGroup(ctx, in.GetGroup().GetID()) {
		return status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	return nil
}

// validateRemediateType returns the appropriate remediate type or the
// NULL DB type if the input is invalid, thus letting the server run
// the policy with the default remediate type.
func validateRemediateType(r string) db.NullRemediateType {
	switch r {
	case "on":
		return db.NullRemediateType{RemediateType: db.RemediateTypeOn, Valid: true}
	case "off":
		return db.NullRemediateType{RemediateType: db.RemediateTypeOff, Valid: true}
	case "dry_run":
		return db.NullRemediateType{RemediateType: db.RemediateTypeDryRun, Valid: true}
	}

	return db.NullRemediateType{Valid: false}
}

// CreatePolicy creates a policy for a group
// nolint: gocyclo
func (s *Server) CreatePolicy(ctx context.Context,
	cpr *mediatorv1.CreatePolicyRequest) (*mediatorv1.CreatePolicyResponse, error) {
	in := cpr.GetPolicy()

	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// If provider doesn't exist, return error
	provider, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:    entityCtx.GetProvider().Name,
		GroupID: entityCtx.GetGroup().ID})
	if err != nil {
		return nil, providerError(fmt.Errorf("provider error: %w", err))
	}

	if err := engine.ValidatePolicy(in); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid policy: %v", err)
	}

	err = engine.TraverseAllRulesForPipeline(in, func(r *mediatorv1.Policy_Rule) error {
		// TODO: This will need to be updated to support
		// the hierarchy tree once that's settled in.
		rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
			Provider: provider.Name,
			GroupID:  entityCtx.GetGroup().GetID(),
			Name:     r.GetType(),
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

		return nil
	})

	if err != nil {
		var violation *engine.RuleValidationError
		if errors.As(err, &violation) {
			log.Printf("error validating rule: %v", violation)
			return nil, util.UserVisibleError(codes.InvalidArgument,
				"policy contained invalid rule '%s': %s", violation.RuleType, violation.Err)
		}

		log.Printf("error getting rule type: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating policy")
	}

	// Now that we know it's valid, let's persist it!
	tx, err := s.store.BeginTransaction()
	if err != nil {
		log.Printf("error starting transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating policy")
	}
	defer s.store.Rollback(tx)

	qtx := s.store.GetQuerierWithTransaction(tx)

	params := db.CreatePolicyParams{
		Provider:  provider.Name,
		GroupID:   entityCtx.GetGroup().GetID(),
		Name:      in.GetName(),
		Remediate: validateRemediateType(in.GetRemediate()),
	}

	// Create policy
	policy, err := qtx.CreatePolicy(ctx, params)
	if err != nil {
		log.Printf("error creating policy: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating policy")
	}

	// Create entity rules entries
	for ent, entRules := range map[mediatorv1.Entity][]*mediatorv1.Policy_Rule{
		mediatorv1.Entity_ENTITY_REPOSITORIES:       in.GetRepository(),
		mediatorv1.Entity_ENTITY_ARTIFACTS:          in.GetArtifact(),
		mediatorv1.Entity_ENTITY_BUILD_ENVIRONMENTS: in.GetBuildEnvironment(),
		mediatorv1.Entity_ENTITY_PULL_REQUESTS:      in.GetPullRequest(),
	} {
		if err := createPolicyRulesForEntity(ctx, ent, &policy, qtx, entRules); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("error committing transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating policy")
	}

	idStr := policy.ID.String()
	in.Id = &idStr
	resp := &mediatorv1.CreatePolicyResponse{
		Policy: in,
	}

	msg, err := reconcilers.NewPolicyInitMessage(entityCtx.Provider.Name, entityCtx.Group.ID)
	if err != nil {
		log.Printf("error creating reconciler event: %v", err)
		// error is non-fatal
		return resp, nil
	}

	// This is a non-fatal error, so we'll just log it and continue with the next ones
	if err := s.evt.Publish(reconcilers.InternalPolicyInitEventTopic, msg); err != nil {
		log.Printf("error publishing reconciler event: %v", err)
	}

	return resp, nil
}

func createPolicyRulesForEntity(
	ctx context.Context,
	entity mediatorv1.Entity,
	policy *db.Policy,
	qtx db.Querier,
	rules []*mediatorv1.Policy_Rule,
) error {
	if rules == nil {
		return nil
	}

	marshalled, err := json.Marshal(rules)
	if err != nil {
		log.Printf("error marshalling %s rules: %v", entity, err)
		return status.Errorf(codes.Internal, "error creating policy")
	}
	_, err = qtx.CreatePolicyForEntity(ctx, db.CreatePolicyForEntityParams{
		PolicyID:        policy.ID,
		Entity:          entities.EntityTypeToDB(entity),
		ContextualRules: marshalled,
	})

	return err
}

// DeletePolicy is a method to delete a policy
func (s *Server) DeletePolicy(ctx context.Context,
	in *mediatorv1.DeletePolicyRequest) (*mediatorv1.DeletePolicyResponse, error) {
	_, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	parsedPolicyID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid policy ID")
	}

	_, err = s.store.GetPolicyByID(ctx, parsedPolicyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "policy not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get policy: %s", err)
	}

	err = s.store.DeletePolicy(ctx, parsedPolicyID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete policy: %s", err)
	}

	return &mediatorv1.DeletePolicyResponse{}, nil
}

// ListPolicies is a method to get all policies for a group
func (s *Server) ListPolicies(ctx context.Context,
	in *mediatorv1.ListPoliciesRequest) (*mediatorv1.ListPoliciesResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	policies, err := s.store.ListPoliciesByGroupID(ctx, entityCtx.Group.ID)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get policies: %s", err)
	}

	var resp mediatorv1.ListPoliciesResponse
	resp.Policies = make([]*mediatorv1.Policy, 0, len(policies))
	for _, policy := range engine.MergeDatabaseListIntoPolicies(policies, entityCtx) {
		resp.Policies = append(resp.Policies, policy)
	}

	return &resp, nil
}

// GetPolicyById is a method to get a policy by id
func (s *Server) GetPolicyById(ctx context.Context,
	in *mediatorv1.GetPolicyByIdRequest) (*mediatorv1.GetPolicyByIdResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	parsedPolicyID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid policy ID")
	}

	policies, err := s.store.GetPolicyByGroupAndID(ctx, db.GetPolicyByGroupAndIDParams{
		GroupID: entityCtx.Group.ID,
		ID:      parsedPolicyID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get policy: %s", err)
	}

	var resp mediatorv1.GetPolicyByIdResponse
	pols := engine.MergeDatabaseGetIntoPolicies(policies, entityCtx)
	if len(pols) == 0 {
		return nil, status.Errorf(codes.NotFound, "policy not found")
	} else if len(pols) > 1 {
		return nil, status.Errorf(codes.Unknown, "failed to get policy: %s", err)
	}

	// This should be only one policy
	for _, policy := range pols {
		resp.Policy = policy
	}

	return &resp, nil
}

func getRuleEvalEntityInfo(
	ctx context.Context,
	store db.Store,
	entityType *db.NullEntities,
	selector *uuid.NullUUID,
	rs db.ListRuleEvaluationStatusByPolicyIdRow,
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

// GetPolicyStatusById is a method to get policy status
// nolint:gocyclo // TODO: Refactor this to be more readable
func (s *Server) GetPolicyStatusById(ctx context.Context,
	in *mediatorv1.GetPolicyStatusByIdRequest) (*mediatorv1.GetPolicyStatusByIdResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	parsedPolicyID, err := uuid.Parse(in.PolicyId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid policy ID")
	}

	dbstat, err := s.store.GetPolicyStatusByIdAndGroup(ctx, db.GetPolicyStatusByIdAndGroupParams{
		GroupID: entityCtx.Group.ID,
		ID:      parsedPolicyID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "policy status not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get policy: %s", err)
	}

	var rulestats []*mediatorv1.RuleEvaluationStatus
	var selector *uuid.NullUUID
	var dbEntity *db.NullEntities

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

	// TODO: Handle retrieving status for other types of entities
	if selector != nil {
		dbrulestat, err := s.store.ListRuleEvaluationStatusByPolicyId(ctx, db.ListRuleEvaluationStatusByPolicyIdParams{
			PolicyID:   parsedPolicyID,
			EntityID:   *selector,
			EntityType: *dbEntity,
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.Unknown, "failed to list rule evaluation status: %s", err)
		}

		rulestats = make([]*mediatorv1.RuleEvaluationStatus, 0, len(dbrulestat))
		for _, rs := range dbrulestat {
			var guidance string
			if rs.EvalStatus == db.EvalStatusTypesFailure || rs.EvalStatus == db.EvalStatusTypesError {
				ruleTypeInfo, err := s.store.GetRuleTypeByID(ctx, rs.RuleTypeID)
				if err != nil {
					log.Printf("error getting rule type info: %v", err)
				} else {
					guidance = ruleTypeInfo.Guidance
				}
			}

			st := &mediatorv1.RuleEvaluationStatus{
				PolicyId:    in.PolicyId,
				RuleId:      rs.RuleTypeID.String(),
				RuleName:    rs.RuleTypeName,
				Entity:      string(rs.Entity),
				Status:      string(rs.EvalStatus),
				Details:     rs.Details,
				EntityInfo:  getRuleEvalEntityInfo(ctx, s.store, dbEntity, selector, rs, entityCtx.GetProvider().Name),
				Guidance:    guidance,
				LastUpdated: timestamppb.New(rs.LastUpdated),
			}

			rulestats = append(rulestats, st)
		}

		// TODO: Add other entities once we have database entries for them
	}

	return &mediatorv1.GetPolicyStatusByIdResponse{
		PolicyStatus: &mediatorv1.PolicyStatus{
			PolicyId:     dbstat.ID.String(),
			PolicyName:   dbstat.Name,
			PolicyStatus: string(dbstat.PolicyStatus),
			LastUpdated:  timestamppb.New(dbstat.LastUpdated),
		},
		RuleEvaluationStatus: rulestats,
	}, nil
}

// GetPolicyStatusByGroup is a method to get policy status for a group
func (s *Server) GetPolicyStatusByGroup(ctx context.Context,
	in *mediatorv1.GetPolicyStatusByGroupRequest) (*mediatorv1.GetPolicyStatusByGroupResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	// read policy status
	dbstats, err := s.store.GetPolicyStatusByGroup(ctx, entityCtx.Group.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "policy statuses not found for group")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get policy status: %s", err)
	}

	res := &mediatorv1.GetPolicyStatusByGroupResponse{
		PolicyStatus: make([]*mediatorv1.PolicyStatus, 0, len(dbstats)),
	}

	for _, dbstat := range dbstats {
		res.PolicyStatus = append(res.PolicyStatus, &mediatorv1.PolicyStatus{
			PolicyId:     dbstat.ID.String(),
			PolicyName:   dbstat.Name,
			PolicyStatus: string(dbstat.PolicyStatus),
		})
	}

	return res, nil
}

// Rule type CRUD

// ListRuleTypes is a method to list all rule types for a given context
func (s *Server) ListRuleTypes(
	ctx context.Context,
	in *mediatorv1.ListRuleTypesRequest,
) (*mediatorv1.ListRuleTypesResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	lrt, err := s.store.ListRuleTypesByProviderAndGroup(ctx, db.ListRuleTypesByProviderAndGroupParams{
		Provider: entityCtx.GetProvider().Name,
		GroupID:  entityCtx.GetGroup().GetID(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get rule types: %s", err)
	}

	resp := &mediatorv1.ListRuleTypesResponse{}

	for idx := range lrt {
		rt := lrt[idx]
		rtpb, err := engine.RuleTypePBFromDB(&rt, entityCtx)
		if err != nil {
			return nil, fmt.Errorf("cannot convert rule type %s to pb: %v", rt.Name, err)
		}

		resp.RuleTypes = append(resp.RuleTypes, rtpb)
	}

	return resp, nil
}

// GetRuleTypeByName is a method to get a rule type by name
func (s *Server) GetRuleTypeByName(
	ctx context.Context,
	in *mediatorv1.GetRuleTypeByNameRequest,
) (*mediatorv1.GetRuleTypeByNameResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	resp := &mediatorv1.GetRuleTypeByNameResponse{}

	rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider: entityCtx.GetProvider().Name,
		GroupID:  entityCtx.GetGroup().GetID(),
		Name:     in.GetName(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get rule type: %s", err)
	}

	rt, err := engine.RuleTypePBFromDB(&rtdb, entityCtx)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule type %s to pb: %v", rtdb.Name, err)
	}

	resp.RuleType = rt

	return resp, nil
}

// GetRuleTypeById is a method to get a rule type by id
func (s *Server) GetRuleTypeById(
	ctx context.Context,
	in *mediatorv1.GetRuleTypeByIdRequest,
) (*mediatorv1.GetRuleTypeByIdResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	resp := &mediatorv1.GetRuleTypeByIdResponse{}

	parsedRuleTypeID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid rule type ID")
	}

	rtdb, err := s.store.GetRuleTypeByID(ctx, parsedRuleTypeID)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get rule type: %s", err)
	}

	rt, err := engine.RuleTypePBFromDB(&rtdb, entityCtx)
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule type %s to pb: %v", rtdb.Name, err)
	}

	resp.RuleType = rt

	return resp, nil
}

// CreateRuleType is a method to create a rule type
func (s *Server) CreateRuleType(
	ctx context.Context,
	crt *mediatorv1.CreateRuleTypeRequest,
) (*mediatorv1.CreateRuleTypeResponse, error) {
	in := crt.GetRuleType()

	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)
	_, err = s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider: entityCtx.GetProvider().Name,
		GroupID:  entityCtx.GetGroup().GetID(),
		Name:     in.GetName(),
	})
	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "rule type %s already exists", in.GetName())
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.Unknown, "failed to get rule type: %s", err)
	}

	if err := in.Validate(); err != nil {
		if errors.Is(err, mediatorv1.ErrInvalidRuleType) {
			return nil, util.UserVisibleError(codes.InvalidArgument, "Couldn't create rule: %s", err)
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid rule type definition: %v", err)
	}

	def, err := util.GetBytesFromProto(in.GetDef())
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule definition to db: %v", err)
	}

	dbrtyp, err := s.store.CreateRuleType(ctx, db.CreateRuleTypeParams{
		Name:        in.GetName(),
		Provider:    entityCtx.GetProvider().Name,
		GroupID:     entityCtx.GetGroup().GetID(),
		Description: in.GetDescription(),
		Definition:  def,
		Guidance:    in.GetGuidance(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to create rule type: %s", err)
	}

	rtypeIDStr := dbrtyp.ID.String()
	in.Id = &rtypeIDStr

	return &mediatorv1.CreateRuleTypeResponse{
		RuleType: in,
	}, nil
}

// UpdateRuleType is a method to update a rule type
func (s *Server) UpdateRuleType(
	ctx context.Context,
	urt *mediatorv1.UpdateRuleTypeRequest,
) (*mediatorv1.UpdateRuleTypeResponse, error) {
	in := urt.GetRuleType()

	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider: entityCtx.GetProvider().Name,
		GroupID:  entityCtx.GetGroup().GetID(),
		Name:     in.GetName(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "rule type %s not found", in.GetName())
		}
		return nil, status.Errorf(codes.Internal, "failed to get rule type: %s", err)
	}

	if err := in.Validate(); err != nil {
		if errors.Is(err, mediatorv1.ErrInvalidRuleType) {
			return nil, util.UserVisibleError(codes.InvalidArgument, "Couldn't update rule: %s", err)
		}
		return nil, status.Errorf(codes.Unavailable, "invalid rule type definition: %s", err)
	}

	def, err := util.GetBytesFromProto(in.GetDef())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot convert rule definition to db: %s", err)
	}

	err = s.store.UpdateRuleType(ctx, db.UpdateRuleTypeParams{
		ID:          rtdb.ID,
		Description: in.GetDescription(),
		Definition:  def,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to create rule type: %s", err)
	}

	return &mediatorv1.UpdateRuleTypeResponse{
		RuleType: in,
	}, nil
}

// DeleteRuleType is a method to delete a rule type
func (s *Server) DeleteRuleType(
	ctx context.Context,
	in *mediatorv1.DeleteRuleTypeRequest,
) (*mediatorv1.DeleteRuleTypeResponse, error) {
	parsedRuleTypeID, err := uuid.Parse(in.GetId())
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid rule type ID")
	}

	// first read rule type by id, so we can get provider
	ruletype, err := s.store.GetRuleTypeByID(ctx, parsedRuleTypeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "rule type %s not found", in.GetId())
		}
		return nil, status.Errorf(codes.Unknown, "failed to get rule type: %s", err)
	}

	prov, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:    ruletype.Provider,
		GroupID: ruletype.GroupID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get provider: %s", err)
	}

	in.Context.Provider = prov.Name

	ctx, err = s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	err = s.store.DeleteRuleType(ctx, parsedRuleTypeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "rule type %s not found", in.GetId())
		}
		return nil, status.Errorf(codes.Unknown, "failed to delete rule type: %s", err)
	}

	return &mediatorv1.DeleteRuleTypeResponse{}, nil
}
