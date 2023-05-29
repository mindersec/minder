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

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CreateUserValidation struct {
	OrganisationId int32  `db:"organisation_id" validate:"required"`
	RoleId         int32  `db:"role_id" validate:"required"`
	GroupId        int32  `db:"group_id" validate:"required"`
	Email          string `db:"email" validate:"required,email"`
	Username       string `db:"username" validate:"required"`
	Password       string `validate:"required,min=8,containsany=!@#?*"`
}

func stringToNullString(s string) *sql.NullString {
	if s == "" {
		return nil
	}
	return &sql.NullString{String: s, Valid: true}
}

// CreateUser is a service for creating an organisation
func (s *Server) CreateUser(ctx context.Context,
	in *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(CreateUserValidation{OrganisationId: in.OrganisationId, RoleId: in.RoleId,
		GroupId: in.GroupId, Email: in.Email, Username: in.Username, Password: in.Password})
	if err != nil {
		return nil, err
	}

	if in.IsProtected == nil {
		isProtected := false
		in.IsProtected = &isProtected
	}

	user, err := s.store.CreateUser(ctx, db.CreateUserParams{OrganisationID: in.OrganisationId, RoleID: in.RoleId,
		GroupID: in.GroupId, Email: in.Email, Username: in.Username, Password: in.Password,
		FirstName: *stringToNullString(*in.FirstName), LastName: *stringToNullString(*in.LastName), IsProtected: *in.IsProtected})
	if err != nil {
		return nil, err
	}

	return &pb.CreateUserResponse{Id: user.ID, OrganisationId: user.OrganisationID, RoleId: user.RoleID, GroupId: user.GroupID,
		Email: user.Email, Username: user.Username, FirstName: &user.FirstName.String, LastName: &user.LastName.String,
		IsProtected: &user.IsProtected, CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt)}, nil

}
