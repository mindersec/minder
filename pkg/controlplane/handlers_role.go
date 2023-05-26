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

	"github.com/stacklok/mediator/internal/role"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateRole is a service for creating an organisation
func (s *Server) CreateRole(ctx context.Context,
	in *pb.CreateRoleRequest) (*pb.CreateRoleResponse, error) {
	r, err := role.CreateRole(ctx, s.store, in.GroupId, in.Name, in.IsAdmin, in.IsProtected)
	if err != nil {
		return nil, err
	}
	return &pb.CreateRoleResponse{Id: r.ID, GroupId: r.GroupID, Name: r.Name, IsAdmin: r.IsAdmin,
		IsProtected: r.IsProtected, CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt)}, nil
}
