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

	"github.com/stacklok/mediator/internal/organisation"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateOrganisation is a service for creating an organisation
func (s *Server) CreateOrganisation(ctx context.Context,
	in *pb.CreateOrganisationRequest) (*pb.CreateOrganisationResponse, error) {
	fmt.Println(s)
	org, err := organisation.CreateOrganisation(ctx, s.store, in.GetName(), in.GetCompany())
	if err != nil {
		return nil, err
	}
	return &pb.CreateOrganisationResponse{Id: org.ID, Name: org.Name,
		Company: org.Company, CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt)}, nil
}
