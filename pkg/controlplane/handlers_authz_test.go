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
	"testing"

	"github.com/golang/mock/gomock"
	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/pkg/auth"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stretchr/testify/assert"
)

func TestIsSuperadminAuthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	request := &pb.GetGroupByIdRequest{GroupId: 1}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      true,
		IsSuperadmin: true,
	})

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetGroupByID(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetGroupById(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestIsNonadminAuthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	request := &pb.CreateRoleRequest{GroupId: 1, Name: "test"}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      false,
		IsSuperadmin: false,
	})

	mockStore := mockdb.NewMockStore(ctrl)
	server := &Server{
		store: mockStore,
	}
	mockStore.EXPECT().CreateRole(ctx, gomock.Any()).Times(0)

	_, err := server.CreateRole(ctx, request)

	if err == nil {
		t.Error("Expected error when user is not authorized, but got nil")
	} else {
		t.Logf("Successfully received error when user is not authorized: %v", err)
	}
}

func TestByResourceUnauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	request := &pb.GetRoleByIdRequest{Id: 1}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      false,
		IsSuperadmin: false,
	})

	mockStore := mockdb.NewMockStore(ctrl)
	server := &Server{
		store: mockStore,
	}
	mockStore.EXPECT().GetRoleByID(ctx, gomock.Any()).Times(1)

	_, err := server.GetRoleById(ctx, request)

	if err == nil {
		t.Error("Expected error when user is not authorized, but got nil")
	} else {
		t.Logf("Successfully received error when user is not authorized: %v", err)
	}
}
