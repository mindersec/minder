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

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

type policyStructure struct {
	YAMLContent string `yaml:"yaml_content"`
}

func validPolicySchema(fl validator.FieldLevel) bool {
	// for the moment we check if it is a valid yaml
	// in the future we could validate against some schema based on policy type
	value, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}

	var temp policyStructure
	err := yaml.Unmarshal([]byte(value), &temp)
	return err == nil
}

// CreatePolicyValidation is a struct for validating the CreatePolicy request
type CreatePolicyValidation struct {
	Provider string        `db:"provider" validate:"required"`
	GroupId  int32         `db:"group_id" validate:"required"`
	Type     pb.PolicyType `db:"type" validate:"required"`
	Policy   string        `db:"policy" validate:"required,validPolicySchema"`
}

func convertToProtoPolicyType(policyType db.PolicyType) pb.PolicyType {
	switch policyType {
	case db.PolicyTypePOLICYTYPEUNSPECIFIED:
		return pb.PolicyType_POLICY_TYPE_UNSPECIFIED
	case db.PolicyTypePOLICYTYPEBRANCHPROTECTION:
		return pb.PolicyType_POLICY_TYPE_BRANCH_PROTECTION
	default:
		return pb.PolicyType_POLICY_TYPE_UNSPECIFIED
	}
}

// CreatePolicy creates a policy for a group
func (s *Server) CreatePolicy(ctx context.Context,
	in *pb.CreatePolicyRequest) (*pb.CreatePolicyResponse, error) {
	validator := validator.New()
	err := validator.RegisterValidation("validPolicySchema", validPolicySchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot register validation: %v", err)
	}

	if in.Provider != auth.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}

	// set default group if not set
	if in.GroupId == 0 {
		group, err := auth.GetDefaultGroup(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot infer group id")
		}
		in.GroupId = group
	}

	err = validator.Struct(CreatePolicyValidation{Provider: in.Provider, GroupId: in.GroupId,
		Type: *in.Type.Enum(), Policy: in.PolicyDefinition})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	policy, err := s.store.CreatePolicy(ctx, db.CreatePolicyParams{Provider: in.Provider, GroupID: in.GroupId,
		PolicyType: db.PolicyType(in.Type.Enum().String()), PolicyDefinition: in.PolicyDefinition})
	if err != nil {
		return nil, err
	}

	return &pb.CreatePolicyResponse{Policy: &pb.PolicyRecord{Id: policy.ID, Provider: policy.Provider, GroupId: policy.GroupID,
		Type: convertToProtoPolicyType(policy.PolicyType), PolicyDefinition: string(policy.PolicyDefinition),
		CreatedAt: timestamppb.New(policy.CreatedAt), UpdatedAt: timestamppb.New(policy.UpdatedAt)}}, nil
}

type deletePolicyValidation struct {
	Id int32 `db:"id" validate:"required"`
}

// DeletePolicy is a method to delete a policy
func (s *Server) DeletePolicy(ctx context.Context,
	in *pb.DeletePolicyRequest) (*pb.DeletePolicyResponse, error) {
	validator := validator.New()
	err := validator.Struct(deletePolicyValidation{Id: in.Id})
	if err != nil {
		return nil, err
	}

	// first check if the policy exists and is not protected
	policy, err := s.store.GetPolicyByID(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, policy.GroupID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	err = s.store.DeletePolicy(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeletePolicyResponse{}, nil
}

// GetPolicies is a method to get all policies for a group
func (s *Server) GetPolicies(ctx context.Context,
	in *pb.GetPoliciesRequest) (*pb.GetPoliciesResponse, error) {

	if in.Provider != auth.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}

	// set default group if not set
	if in.GroupId == 0 {
		group, err := auth.GetDefaultGroup(ctx)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "cannot infer group id")
		}
		in.GroupId = group
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// define default values for limit and offset
	if in.Limit == nil || *in.Limit == -1 {
		in.Limit = new(int32)
		*in.Limit = PaginationLimit
	}
	if in.Offset == nil {
		in.Offset = new(int32)
		*in.Offset = 0
	}

	policies, err := s.store.ListPoliciesByGroupID(ctx, db.ListPoliciesByGroupIDParams{
		Provider: in.Provider,
		GroupID:  in.GroupId,
		Limit:    *in.Limit,
		Offset:   *in.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get policies: %s", err)
	}

	var resp pb.GetPoliciesResponse
	resp.Policies = make([]*pb.PolicyRecord, 0, len(policies))
	for _, policy := range policies {
		resp.Policies = append(resp.Policies, &pb.PolicyRecord{
			Id:               policy.ID,
			Provider:         policy.Provider,
			GroupId:          policy.GroupID,
			Type:             convertToProtoPolicyType(policy.PolicyType),
			PolicyDefinition: string(policy.PolicyDefinition),
			CreatedAt:        timestamppb.New(policy.CreatedAt),
			UpdatedAt:        timestamppb.New(policy.UpdatedAt),
		})
	}

	return &resp, nil
}

// GetPolicyById is a method to get a policy by id
func (s *Server) GetPolicyById(ctx context.Context,
	in *pb.GetPolicyByIdRequest) (*pb.GetPolicyByIdResponse, error) {
	if in.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "policy id is required")
	}

	policy, err := s.store.GetPolicyByID(ctx, in.Id)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get policy: %s", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, policy.GroupID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	var resp pb.GetPolicyByIdResponse
	resp.Policy = &pb.PolicyRecord{
		Id:               policy.ID,
		Provider:         policy.Provider,
		GroupId:          policy.GroupID,
		Type:             convertToProtoPolicyType(policy.PolicyType),
		PolicyDefinition: string(policy.PolicyDefinition),
		CreatedAt:        timestamppb.New(policy.CreatedAt),
		UpdatedAt:        timestamppb.New(policy.UpdatedAt),
	}
	return &resp, nil
}
