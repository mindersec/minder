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

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/mediator/internal/engine"
	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

// authAndContextValidation is a helper function to initialize entity context info and validate input
// It also sets up the needed information in the `in` entity context that's needed for the rest of the flow
// Note that this also does an authorization check.
func (s *Server) authAndContextValidation(ctx context.Context, inout *pb.Context) (context.Context, error) {
	if inout == nil {
		return ctx, fmt.Errorf("context cannot be nil")
	}

	if inout.Provider != ghclient.Github {
		return ctx, fmt.Errorf("provider not supported: %s", inout.Provider)
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
func (s *Server) ensureDefaultGroupForContext(ctx context.Context, inout *pb.Context) error {
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

// CreatePolicy creates a policy for a group
func (s *Server) CreatePolicy(ctx context.Context,
	cpr *pb.CreatePolicyRequest) (*pb.CreatePolicyResponse, error) {
	in := cpr.GetPolicy()

	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	if err := engine.ValidatePolicy(in); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid policy: %v", err)
	}

	err = engine.TraverseAllRulesForPipeline(in, func(r *pb.PipelinePolicy_Rule) error {
		// TODO: This will need to be updated to support
		// the hierarchy tree once that's settled in.
		rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
			Provider: in.GetContext().GetProvider(),
			GroupID:  entityCtx.GetGroup().GetID(),
			Name:     r.GetType(),
		})
		if err != nil {
			return fmt.Errorf("error creating policy")
		}

		rtyppb, err := engine.RuleTypePBFromDB(&rtdb, entityCtx)
		if err != nil {
			return fmt.Errorf("cannot convert rule type %s to pb: %w", rtdb.Name, err)
		}

		rval, err := engine.NewRuleValidator(rtyppb)
		if err != nil {
			return fmt.Errorf("error creating rule validator: %w", err)
		}

		if err := rval.ValidateAgainstSchema(r.Def); err != nil {
			return fmt.Errorf("error validating rule: %w", err)
		}

		return nil
	})

	if err != nil {
		var violation *engine.RuleValidationError
		if errors.As(err, &violation) {
			return nil, status.Errorf(codes.InvalidArgument, "policy contained invalid rule: %s", violation.RuleType)
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

	// Create policy
	policy, err := qtx.CreatePolicy(ctx, db.CreatePolicyParams{
		Provider: in.GetContext().GetProvider(),
		GroupID:  entityCtx.GetGroup().GetID(),
		Name:     in.GetName(),
	})
	if err != nil {
		log.Printf("error creating policy: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating policy")
	}

	// Create entity rules entries
	for ent, entRules := range map[engine.EntityType][]*pb.PipelinePolicy_ContextualRuleSet{
		engine.RepositoryEntity:       in.GetRepository(),
		engine.ArtifactEntity:         in.GetBuildEnvironment(),
		engine.BuildEnvironmentEntity: in.GetArtifact(),
	} {
		if err := createPolicyRulesForEntity(ctx, ent, &policy, qtx, entRules); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("error committing transaction: %v", err)
		return nil, status.Errorf(codes.Internal, "error creating policy")
	}

	in.Id = &policy.ID
	resp := &pb.CreatePolicyResponse{
		Policy: in,
	}

	s.publishPolicyInitEvent(in, entityCtx)

	return resp, nil
}

func (s *Server) publishPolicyInitEvent(pol *pb.PipelinePolicy, ectx *engine.EntityContext) {
	log.Printf("publishing init event for policy: %s", pol.GetName())
	evt := &engine.InitEvent{
		Policy: *pol.Id,
		Group:  ectx.Group.ID,
	}

	evtStr, err := json.Marshal(evt)
	// This is a non-fatal error, so we'll just log it
	// and return the policy.
	if err != nil {
		log.Printf("error marshalling init event: %v", err)
		return
	}

	msg := message.NewMessage(uuid.New().String(), evtStr)
	msg.Metadata.Set("provider", ectx.Provider)

	// This is a non-fatal error, so we'll just log it
	// and continue
	if err := s.evt.Publish(engine.InternalInitEventTopic, msg); err != nil {
		log.Printf("error publishing init event: %v", err)
	}
}

func createPolicyRulesForEntity(
	ctx context.Context,
	entity engine.EntityType,
	policy *db.Policy,
	qtx db.Querier,
	rules []*pb.PipelinePolicy_ContextualRuleSet,
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
		Entity:          db.Entities(entity.String()),
		ContextualRules: marshalled,
	})

	return err
}

// DeletePolicy is a method to delete a policy
func (s *Server) DeletePolicy(ctx context.Context,
	in *pb.DeletePolicyRequest) (*pb.DeletePolicyResponse, error) {
	_, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	if in.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy id is required")
	}

	_, err = s.store.GetPolicyByID(ctx, in.Id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "policy not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get policy: %s", err)
	}

	err = s.store.DeletePolicy(ctx, in.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete policy: %s", err)
	}

	return &pb.DeletePolicyResponse{}, nil
}

// ListPolicies is a method to get all policies for a group
func (s *Server) ListPolicies(ctx context.Context,
	in *pb.ListPoliciesRequest) (*pb.ListPoliciesResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	policies, err := s.store.ListPoliciesByGroupID(ctx, entityCtx.Group.ID)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get policies: %s", err)
	}

	var resp pb.ListPoliciesResponse
	resp.Policies = make([]*pb.PipelinePolicy, 0, len(policies))
	for _, policy := range engine.MergeDatabaseListIntoPolicies(policies, entityCtx) {
		resp.Policies = append(resp.Policies, policy)
	}

	return &resp, nil
}

// GetPolicyById is a method to get a policy by id
func (s *Server) GetPolicyById(ctx context.Context,
	in *pb.GetPolicyByIdRequest) (*pb.GetPolicyByIdResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	if in.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy id is required")
	}

	policies, err := s.store.GetPolicyByGroupAndID(ctx, db.GetPolicyByGroupAndIDParams{
		GroupID: entityCtx.Group.ID,
		ID:      in.Id,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get policy: %s", err)
	}

	var resp pb.GetPolicyByIdResponse
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

// GetPolicyStatusById is a method to get policy status
func (s *Server) GetPolicyStatusById(ctx context.Context,
	in *pb.GetPolicyStatusByIdRequest) (*pb.GetPolicyStatusByIdResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	if in.PolicyId == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy id is required")
	}

	dbstat, err := s.store.GetPolicyStatusByIdAndGroup(ctx, db.GetPolicyStatusByIdAndGroupParams{
		GroupID: entityCtx.Group.ID,
		ID:      in.PolicyId,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "policy status not found")
		}
		return nil, status.Errorf(codes.Unknown, "failed to get policy: %s", err)
	}

	var rulestats []*pb.RuleEvaluationStatus

	if in.All {

		dbrulestat, err := s.store.ListRuleEvaluationStatusForRepositoriesByPolicyId(ctx, in.PolicyId)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.Unknown, "failed to get policy: %s", err)
		}

		rulestats = make([]*pb.RuleEvaluationStatus, 0, len(dbrulestat))
		for _, rs := range dbrulestat {
			st := &pb.RuleEvaluationStatus{
				PolicyId: in.PolicyId,
				RuleId:   rs.RuleTypeID,
				RuleName: rs.RuleTypeName,
				Entity:   engine.RepositoryEntity.String(),
				Status:   string(rs.EvalStatus),
				Details:  rs.Details,
				EntityInfo: map[string]string{
					"repository_id": fmt.Sprintf("%d", rs.RepositoryID.Int32),
					"repo_name":     rs.RepoName,
					"repo_owner":    rs.RepoOwner,
					"provider":      rs.Provider,
				},
			}

			rulestats = append(rulestats, st)
		}

		// TODO: Add other entities once we have database entries for them
	}

	res := &pb.GetPolicyStatusByIdResponse{
		PolicyStatus: &pb.PolicyStatus{
			PolicyId:     dbstat.ID,
			PolicyName:   dbstat.Name,
			PolicyStatus: string(dbstat.PolicyStatus),
		},
		RuleEvaluationStatus: rulestats,
	}

	return res, nil

}

// GetPolicyStatusByGroup is a method to get policy status for a group
func (s *Server) GetPolicyStatusByGroup(ctx context.Context,
	in *pb.GetPolicyStatusByGroupRequest) (*pb.GetPolicyStatusByGroupResponse, error) {
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

	res := &pb.GetPolicyStatusByGroupResponse{
		PolicyStatus: make([]*pb.PolicyStatus, 0, len(dbstats)),
	}

	for _, dbstat := range dbstats {
		res.PolicyStatus = append(res.PolicyStatus, &pb.PolicyStatus{
			PolicyId:     dbstat.ID,
			PolicyName:   dbstat.Name,
			PolicyStatus: string(dbstat.PolicyStatus),
		})
	}

	return res, nil
}

// Rule type CRUD

// ListRuleTypes is a method to list all rule types for a given context
func (s *Server) ListRuleTypes(ctx context.Context, in *pb.ListRuleTypesRequest) (*pb.ListRuleTypesResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	lrt, err := s.store.ListRuleTypesByProviderAndGroup(ctx, db.ListRuleTypesByProviderAndGroupParams{
		Provider: entityCtx.GetProvider(),
		GroupID:  entityCtx.GetGroup().GetID(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get rule types: %s", err)
	}

	resp := &pb.ListRuleTypesResponse{}

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
func (s *Server) GetRuleTypeByName(ctx context.Context, in *pb.GetRuleTypeByNameRequest) (*pb.GetRuleTypeByNameResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	resp := &pb.GetRuleTypeByNameResponse{}

	rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider: entityCtx.GetProvider(),
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
func (s *Server) GetRuleTypeById(ctx context.Context, in *pb.GetRuleTypeByIdRequest) (*pb.GetRuleTypeByIdResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	resp := &pb.GetRuleTypeByIdResponse{}

	rtdb, err := s.store.GetRuleTypeByID(ctx, in.GetId())
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
func (s *Server) CreateRuleType(ctx context.Context, crt *pb.CreateRuleTypeRequest) (*pb.CreateRuleTypeResponse, error) {
	in := crt.GetRuleType()

	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)
	fmt.Println("entity is")
	fmt.Println(entityCtx)
	fmt.Println(entityCtx.GetGroup().GetID())

	_, err = s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider: entityCtx.GetProvider(),
		GroupID:  entityCtx.GetGroup().GetID(),
		Name:     in.GetName(),
	})
	if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "rule type %s already exists", in.GetName())
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.Unknown, "failed to get rule type: %s", err)
	}

	if err := engine.ValidateRuleTypeDefinition(in.GetDef()); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rule type definition: %v", err)
	}

	def, err := engine.DBRuleDefFromPB(in.GetDef())
	if err != nil {
		return nil, fmt.Errorf("cannot convert rule definition to db: %v", err)
	}

	_, err = s.store.CreateRuleType(ctx, db.CreateRuleTypeParams{
		Name:       in.GetName(),
		Provider:   entityCtx.GetProvider(),
		GroupID:    entityCtx.GetGroup().GetID(),
		Definition: def,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to create rule type: %s", err)
	}

	return &pb.CreateRuleTypeResponse{
		RuleType: in,
	}, nil
}

// UpdateRuleType is a method to update a rule type
func (s *Server) UpdateRuleType(ctx context.Context, urt *pb.UpdateRuleTypeRequest) (*pb.UpdateRuleTypeResponse, error) {
	in := urt.GetRuleType()

	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	entityCtx := engine.EntityFromContext(ctx)

	rtdb, err := s.store.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider: entityCtx.GetProvider(),
		GroupID:  entityCtx.GetGroup().GetID(),
		Name:     in.GetName(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "rule type %s not found", in.GetName())
		}
		return nil, status.Errorf(codes.Internal, "failed to get rule type: %s", err)
	}

	if err := engine.ValidateRuleTypeDefinition(in.GetDef()); err != nil {
		return nil, status.Errorf(codes.Unavailable, "invalid rule type definition: %s", err)
	}

	def, err := engine.DBRuleDefFromPB(in.GetDef())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot convert rule definition to db: %s", err)
	}

	err = s.store.UpdateRuleType(ctx, db.UpdateRuleTypeParams{
		ID:         rtdb.ID,
		Definition: def,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to create rule type: %s", err)
	}

	return &pb.UpdateRuleTypeResponse{
		RuleType: in,
	}, nil
}

// DeleteRuleType is a method to delete a rule type
func (s *Server) DeleteRuleType(ctx context.Context, in *pb.DeleteRuleTypeRequest) (*pb.DeleteRuleTypeResponse, error) {
	ctx, err := s.authAndContextValidation(ctx, in.GetContext())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error ensuring default group: %v", err)
	}

	err = s.store.DeleteRuleType(ctx, in.GetId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "rule type %d not found", in.GetId())
		}
		return nil, status.Errorf(codes.Unknown, "failed to delete rule type: %s", err)
	}

	return &pb.DeleteRuleTypeResponse{}, nil
}
