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
	"path/filepath"

	"embed"

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	github "github.com/stacklok/mediator/pkg/providers/github"

	"github.com/stacklok/mediator/internal/util"
	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

//go:embed policy_types/*
var embeddedFiles embed.FS

func readPolicyTypeSchema(provider string, policyType string, version string) (string, error) {
	filePath := filepath.Join("policy_types", provider, policyType, version, "schema.json")

	// Read the file contents
	schema, err := embeddedFiles.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return "", err
	}
	return string(schema), nil
}

func readDefaultPolicyTypeSchema(provider string, policyType string, version string) (string, error) {
	filePath := filepath.Join("policy_types", provider, policyType, version, "default.yaml")

	// Read the file contents
	schema, err := embeddedFiles.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return "", err
	}
	return string(schema), nil
}

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
	Provider string `db:"provider" validate:"required"`
	GroupId  int32  `db:"group_id" validate:"required"`
	Type     string `db:"type" validate:"required"`
	Policy   string `db:"policy" validate:"required,validPolicySchema"`
}

// CreatePolicy creates a policy for a group
// nolint: gocyclo
func (s *Server) CreatePolicy(ctx context.Context,
	in *pb.CreatePolicyRequest) (*pb.CreatePolicyResponse, error) {
	validator := validator.New()
	err := validator.RegisterValidation("validPolicySchema", validPolicySchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot register validation: %v", err)
	}

	if in.Provider != github.Github {
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
		Type: in.Type, Policy: in.PolicyDefinition})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// check if type is valid
	policies, err := s.store.GetPolicyTypes(ctx, in.Provider)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get policy types: %v", err)
	}

	policyType := db.PolicyType{}
	for _, policy := range policies {
		if policy.PolicyType == in.Type {
			policyType = policy
			break
		}
	}
	if policyType == (db.PolicyType{}) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid policy type: %v", in.Type)
	}

	// convert yaml to json
	jsonData, err := util.ConvertYamlToJson(in.PolicyDefinition)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid policy definition: %v", err)
	}

	// read schema
	jsonSchema, err := readPolicyTypeSchema(in.Provider, in.Type, policyType.Version)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot read policy type schema: %v", err)
	}

	// validate against json schema
	schemaLoader := gojsonschema.NewStringLoader(jsonSchema)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create json schema: %v", err)
	}
	documentLoader := gojsonschema.NewStringLoader(string(jsonData))
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot validate json schema: %v", err)
	}

	if !result.Valid() {
		return nil, status.Errorf(codes.InvalidArgument, "invalid policy definition: %v", result.Errors())
	}

	policy, err := s.store.CreatePolicy(ctx, db.CreatePolicyParams{Provider: in.Provider, GroupID: in.GroupId,
		PolicyType: policyType.ID, PolicyDefinition: jsonData})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot create policy: %v", err)
	}

	// convert returned policy to yaml
	yamlStr, err := util.ConvertJsonToYaml(policy.PolicyDefinition)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot extract policy information: %v", err)
	}

	return &pb.CreatePolicyResponse{Policy: &pb.PolicyRecord{Id: policy.ID, Provider: policy.Provider, GroupId: policy.GroupID,
		Type: in.Type, PolicyDefinition: yamlStr,
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

	if in.Provider != github.Github {
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
		yamlStr, err := util.ConvertJsonToYaml(policy.PolicyDefinition)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot extract policy information: %v", err)
		}

		resp.Policies = append(resp.Policies, &pb.PolicyRecord{
			Id:               policy.ID,
			Provider:         policy.Provider,
			GroupId:          policy.GroupID,
			Type:             policy.PolicyTypeName.String,
			PolicyDefinition: yamlStr,
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

	// convert returned policy to yaml
	yamlStr, err := util.ConvertJsonToYaml(policy.PolicyDefinition)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot extract policy information: %v", err)
	}

	var resp pb.GetPolicyByIdResponse
	resp.Policy = &pb.PolicyRecord{
		Id:               policy.ID,
		Provider:         policy.Provider,
		GroupId:          policy.GroupID,
		Type:             policy.PolicyTypeName.String,
		PolicyDefinition: yamlStr,
		CreatedAt:        timestamppb.New(policy.CreatedAt),
		UpdatedAt:        timestamppb.New(policy.UpdatedAt),
	}
	return &resp, nil
}

// GetPolicyTypes is a method to get all policy types
func (s *Server) GetPolicyTypes(ctx context.Context, in *pb.GetPolicyTypesRequest) (*pb.GetPolicyTypesResponse, error) {
	if in.Provider != github.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}
	types, err := s.store.GetPolicyTypes(ctx, in.Provider)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get policy types: %s", err)
	}

	var resp pb.GetPolicyTypesResponse
	resp.PolicyTypes = make([]*pb.PolicyTypeRecord, 0, len(types))
	for _, policyType := range types {
		// in list, we do not return json schema to optimize the response
		resp.PolicyTypes = append(resp.PolicyTypes, &pb.PolicyTypeRecord{
			Id: policyType.ID, Provider: policyType.Provider,
			PolicyType:  policyType.PolicyType,
			Description: &policyType.Description.String,
			JsonSchema:  "", DefaultSchema: "",
			Version: policyType.Version, CreatedAt: timestamppb.New(policyType.CreatedAt),
			UpdatedAt: timestamppb.New(policyType.UpdatedAt),
		})
	}

	return &resp, nil
}

// GetPolicyType is a method to get a policy type by id
func (s *Server) GetPolicyType(ctx context.Context, in *pb.GetPolicyTypeRequest) (*pb.GetPolicyTypeResponse, error) {
	if in.Provider != github.Github {
		return nil, status.Errorf(codes.InvalidArgument, "provider not supported: %v", in.Provider)
	}
	policyType, err := s.store.GetPolicyType(ctx, db.GetPolicyTypeParams{Provider: in.Provider, PolicyType: in.Type})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get policy types: %s", err)
	}
	schema, err := readPolicyTypeSchema(policyType.Provider, policyType.PolicyType, policyType.Version)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get policy type schemas: %s", err)
	}

	default_schema, err := readDefaultPolicyTypeSchema(policyType.Provider, policyType.PolicyType, policyType.Version)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get default policy schemas: %s", err)
	}

	return &pb.GetPolicyTypeResponse{PolicyType: &pb.PolicyTypeRecord{Id: policyType.ID, Provider: policyType.Provider,
		PolicyType: policyType.PolicyType, Description: &policyType.Description.String,
		JsonSchema: schema, DefaultSchema: default_schema,
		Version: policyType.Version, CreatedAt: timestamppb.New(policyType.CreatedAt),
		UpdatedAt: timestamppb.New(policyType.UpdatedAt)}}, nil
}
