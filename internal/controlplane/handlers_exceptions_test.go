// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/engine/engcontext"
	mockentitysvc "github.com/mindersec/minder/internal/entities/service/mock"
	exceptionSvc "github.com/mindersec/minder/internal/exception/service"
	mockexceptionsvc "github.com/mindersec/minder/internal/exception/service/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func exceptionTestContext(projectID uuid.UUID) context.Context {
	return engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
		Project: engcontext.Project{ID: projectID},
	})
}

func TestListExceptions(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	entityID := uuid.New()
	exceptionID := uuid.New()
	createdAt := time.Now()
	expiresAt := createdAt.Add(time.Hour)

	tests := []struct {
		name      string
		setup     func(*mockexceptionsvc.MockService, *mockentitysvc.MockEntityService)
		wantCount int
		wantErr   bool
	}{
		{
			name: "success",
			setup: func(exceptionMock *mockexceptionsvc.MockService, entityMock *mockentitysvc.MockEntityService) {
				exceptionMock.EXPECT().
					List(gomock.Any(), projectID).
					Return([]exceptionSvc.Exception{
						{
							ID:          exceptionID,
							EntityID:    entityID,
							ProfileName: "test-profile",
							RuleName:    "test-rule",
							Reason:      "test reason",
							ExpiresAt:   expiresAt,
							CreatedAt:   createdAt,
						},
					}, nil)

				entityMock.EXPECT().
					GetEntityByID(gomock.Any(), entityID, projectID).
					Return(&pb.EntityInstance{Id: entityID.String()}, nil)
			},
			wantCount: 1,
		},
		{
			name: "empty list",
			setup: func(exceptionMock *mockexceptionsvc.MockService, _ *mockentitysvc.MockEntityService) {
				exceptionMock.EXPECT().
					List(gomock.Any(), projectID).
					Return([]exceptionSvc.Exception{}, nil)
			},
			wantCount: 0,
		},
		{
			name: "service error",
			setup: func(exceptionMock *mockexceptionsvc.MockService, _ *mockentitysvc.MockEntityService) {
				exceptionMock.EXPECT().
					List(gomock.Any(), projectID).
					Return(nil, errors.New("list failed"))
			},
			wantErr: true,
		},
		{
			name: "entity lookup error",
			setup: func(exceptionMock *mockexceptionsvc.MockService, entityMock *mockentitysvc.MockEntityService) {
				exceptionMock.EXPECT().
					List(gomock.Any(), projectID).
					Return([]exceptionSvc.Exception{{EntityID: entityID}}, nil)

				entityMock.EXPECT().
					GetEntityByID(gomock.Any(), entityID, projectID).
					Return(nil, errors.New("entity lookup failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			exceptionMock := mockexceptionsvc.NewMockService(ctrl)
			entityMock := mockentitysvc.NewMockEntityService(ctrl)

			tt.setup(exceptionMock, entityMock)

			server := &Server{
				exceptions:    exceptionMock,
				entityService: entityMock,
			}

			resp, err := server.ListExceptions(
				exceptionTestContext(projectID),
				&pb.ListExceptionsRequest{},
			)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Len(t, resp.GetExceptions(), tt.wantCount)

			if tt.wantCount == 1 {
				got := resp.GetExceptions()[0]
				assert.Equal(t, exceptionID.String(), got.GetId())
				assert.Equal(t, entityID.String(), got.GetEntity().GetId())
				assert.Equal(t, "test-profile", got.GetProfileName())
				assert.Equal(t, "test-rule", got.GetRuleName())
				assert.Equal(t, "test reason", got.GetReason())
				require.NotNil(t, got.GetExpiresAt())
				require.NotNil(t, got.GetCreatedAt())
				assert.True(t, got.GetExpiresAt().AsTime().Equal(expiresAt))
				assert.True(t, got.GetCreatedAt().AsTime().Equal(createdAt))
			}
		})
	}
}

func TestCreateException(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	entityID := uuid.New()
	exceptionID := uuid.New()
	createdAt := time.Now()
	expiresAt := createdAt.Add(time.Hour)
	reason := ""

	tests := []struct {
		name     string
		request  *pb.CreateExceptionRequest
		setup    func(*mockexceptionsvc.MockService, *mockentitysvc.MockEntityService)
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name: "success with explicitly empty reason",
			request: &pb.CreateExceptionRequest{
				EntityId:    entityID.String(),
				ProfileName: "test-profile",
				RuleName:    "test-rule",
				ExpiresAt:   timestamppb.New(expiresAt),
				Reason:      &reason,
			},
			setup: func(exceptionMock *mockexceptionsvc.MockService, entityMock *mockentitysvc.MockEntityService) {
				exceptionMock.EXPECT().
					Create(gomock.Any(), gomock.Cond(func(got exceptionSvc.Exception) bool {
						return got.EntityID == entityID &&
							got.ProfileName == "test-profile" &&
							got.RuleName == "test-rule" &&
							got.Reason == "" &&
							got.ReasonProvided &&
							got.ExpiresAt.Equal(expiresAt)
					})).
					Return(&exceptionSvc.Exception{
						ID:          exceptionID,
						EntityID:    entityID,
						ProfileName: "test-profile",
						RuleName:    "test-rule",
						Reason:      "",
						ExpiresAt:   expiresAt,
						CreatedAt:   createdAt,
					}, nil)

				entityMock.EXPECT().
					GetEntityByID(gomock.Any(), entityID, projectID).
					Return(&pb.EntityInstance{Id: entityID.String()}, nil)
			},
		},
		{
			name: "invalid entity ID",
			request: &pb.CreateExceptionRequest{
				EntityId:  "not-a-uuid",
				ExpiresAt: timestamppb.New(expiresAt),
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "missing expiry",
			request: &pb.CreateExceptionRequest{
				EntityId: entityID.String(),
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "invalid expiry",
			request: &pb.CreateExceptionRequest{
				EntityId:  entityID.String(),
				ExpiresAt: &timestamppb.Timestamp{Seconds: 253402300800},
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "service error",
			request: &pb.CreateExceptionRequest{
				EntityId:  entityID.String(),
				ExpiresAt: timestamppb.New(expiresAt),
			},
			setup: func(exceptionMock *mockexceptionsvc.MockService, _ *mockentitysvc.MockEntityService) {
				exceptionMock.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("create failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			exceptionMock := mockexceptionsvc.NewMockService(ctrl)
			entityMock := mockentitysvc.NewMockEntityService(ctrl)

			if tt.setup != nil {
				tt.setup(exceptionMock, entityMock)
			}

			server := &Server{
				exceptions:    exceptionMock,
				entityService: entityMock,
			}

			resp, err := server.CreateException(
				exceptionTestContext(projectID),
				tt.request,
			)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, resp)
				if tt.wantCode != codes.OK {
					assert.Equal(t, tt.wantCode, status.Code(err))
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.NotNil(t, resp.GetException())
			assert.Equal(t, exceptionID.String(), resp.GetException().GetId())
			assert.Equal(t, entityID.String(), resp.GetException().GetEntity().GetId())
			assert.Empty(t, resp.GetException().GetReason())
			require.NotNil(t, resp.GetException().GetExpiresAt())
			require.NotNil(t, resp.GetException().GetCreatedAt())
			assert.True(t, resp.GetException().GetExpiresAt().AsTime().Equal(expiresAt))
			assert.True(t, resp.GetException().GetCreatedAt().AsTime().Equal(createdAt))
		})
	}
}

func TestDeleteException(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	exceptionID := uuid.New()

	tests := []struct {
		name     string
		request  *pb.DeleteExceptionRequest
		setup    func(*mockexceptionsvc.MockService)
		wantErr  bool
		wantCode codes.Code
	}{
		{
			name: "success",
			request: &pb.DeleteExceptionRequest{
				Id: exceptionID.String(),
			},
			setup: func(exceptionMock *mockexceptionsvc.MockService) {
				exceptionMock.EXPECT().
					Delete(gomock.Any(), exceptionID, projectID).
					Return(nil)
			},
		},
		{
			name: "invalid exception ID",
			request: &pb.DeleteExceptionRequest{
				Id: "not-a-uuid",
			},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "service returns not found",
			request: &pb.DeleteExceptionRequest{
				Id: exceptionID.String(),
			},
			setup: func(exceptionMock *mockexceptionsvc.MockService) {
				exceptionMock.EXPECT().
					Delete(gomock.Any(), exceptionID, projectID).
					Return(status.Error(codes.NotFound, "exception not found"))
			},
			wantErr:  true,
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			exceptionMock := mockexceptionsvc.NewMockService(ctrl)

			if tt.setup != nil {
				tt.setup(exceptionMock)
			}

			server := &Server{
				exceptions: exceptionMock,
			}

			resp, err := server.DeleteException(
				exceptionTestContext(projectID),
				tt.request,
			)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, resp)
				assert.Equal(t, tt.wantCode, status.Code(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
		})
	}
}
