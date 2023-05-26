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
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CreateOrganisationValidation struct {
	Name    string `db:"name" validate:"required"`
	Company string `db:"company" validate:"required"`
}

// CreateOrganisation is a service for creating an organisation
func (s *Server) CreateOrganisation(ctx context.Context,
	in *pb.CreateOrganisationRequest) (*pb.CreateOrganisationResponse, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(CreateOrganisationValidation{Name: in.Name, Company: in.Company})
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
