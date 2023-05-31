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

// CreateRoleValidation is a struct for validating the CreateRole request
type CreateRoleValidation struct {
	GroupId int32  `db:"group_id" validate:"required"`
	Name    string `db:"name" validate:"required"`
}

// CreateRole is a service for creating an organisation
func (s *Server) CreateRole(ctx context.Context,
	in *pb.CreateRoleRequest) (*pb.CreateRoleResponse, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(CreateRoleValidation{GroupId: in.GroupId, Name: in.Name})
	if err != nil {
		return nil, err
	}

	if in.IsAdmin == nil {
		isAdmin := false
		in.IsAdmin = &isAdmin
	}

	if in.IsProtected == nil {
		isProtected := false
		in.IsProtected = &isProtected
	}

	role, err := s.store.CreateRole(ctx, db.CreateRoleParams{GroupID: in.GroupId,
		Name: in.Name, IsAdmin: *in.IsAdmin, IsProtected: *in.IsProtected})
	if err != nil {
		return nil, err
	}

	return &pb.CreateRoleResponse{Id: role.ID, Name: role.Name,
		IsAdmin: role.IsAdmin, IsProtected: role.IsProtected,
		GroupId:   role.GroupID,
		CreatedAt: timestamppb.New(role.CreatedAt),
		UpdatedAt: timestamppb.New(role.UpdatedAt)}, nil

}
