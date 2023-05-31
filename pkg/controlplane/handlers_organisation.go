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
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type createOrganisationValidation struct {
	Name    string `db:"name" validate:"required"`
	Company string `db:"company" validate:"required"`
}

// CreateOrganisation is a service for creating an organisation
func (s *Server) CreateOrganisation(ctx context.Context,
	in *pb.CreateOrganisationRequest) (*pb.CreateOrganisationResponse, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(createOrganisationValidation{Name: in.Name, Company: in.Company})
	if err != nil {
		return nil, err
	}

	org, err := s.store.CreateOrganisation(ctx, db.CreateOrganisationParams{Name: in.Name, Company: in.Company})
	if err != nil {
		return nil, err
	}

	return &pb.CreateOrganisationResponse{Id: org.ID, Name: org.Name,
		Company: org.Company, CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt)}, nil
}

// GetOrganisations is a service for getting a list of organisations
func (s *Server) GetOrganisations(ctx context.Context,
	in *pb.GetOrganisationsRequest) (*pb.GetOrganisationsResponse, error) {

	// define default values for limit and offset
	if in.Limit == nil || *in.Limit == -1 {
		in.Limit = new(int32)
		*in.Limit = PaginationLimit
	}
	if in.Offset == nil {
		in.Offset = new(int32)
		*in.Offset = 0
	}

	orgs, err := s.store.ListOrganisations(ctx, db.ListOrganisationsParams{
		Limit:  *in.Limit,
		Offset: *in.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	var resp pb.GetOrganisationsResponse
	resp.Organisations = make([]*pb.OrganisationRecord, 0, len(orgs))
	for _, org := range orgs {
		resp.Organisations = append(resp.Organisations, &pb.OrganisationRecord{
			Id:        org.ID,
			Name:      org.Name,
			Company:   org.Company,
			CreatedAt: timestamppb.New(org.CreatedAt),
			UpdatedAt: timestamppb.New(org.UpdatedAt),
		})
	}

	return &resp, nil
}

// GetOrganisation is a service for getting an organisation
func (s *Server) GetOrganisation(ctx context.Context,
	in *pb.GetOrganisationRequest) (*pb.GetOrganisationResponse, error) {
	if in.GetOrganisationId() <= 0 {
		return nil, fmt.Errorf("organisation id is required")
	}

	org, err := s.store.GetOrganisation(ctx, in.OrganisationId)
	if err != nil {
		return nil, fmt.Errorf("failed to get organisation: %w", err)
	}

	var resp pb.GetOrganisationResponse
	resp.Organisation = &pb.OrganisationRecord{
		Id:        org.ID,
		Name:      org.Name,
		Company:   org.Company,
		CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt),
	}

	return &resp, nil
}

// GetOrganisationByName is a service for getting an organisation
func (s *Server) GetOrganisationByName(ctx context.Context,
	in *pb.GetOrganisationByNameRequest) (*pb.GetOrganisationByNameResponse, error) {
	if in.GetName() == "" {
		return nil, fmt.Errorf("organisation name is required")
	}

	org, err := s.store.GetOrganisationByName(ctx, in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get organisation: %w", err)
	}

	var resp pb.GetOrganisationByNameResponse
	resp.Organisation = &pb.OrganisationRecord{
		Id:        org.ID,
		Name:      org.Name,
		Company:   org.Company,
		CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt),
	}

	return &resp, nil
}
