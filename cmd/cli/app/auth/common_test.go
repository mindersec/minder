// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/cmd/cli/app"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

type mockUserServiceClient struct {
	minderv1.UserServiceClient
	getUserFunc func(ctx context.Context, in *minderv1.GetUserRequest, opts ...grpc.CallOption) (*minderv1.GetUserResponse, error)
}

func (m *mockUserServiceClient) GetUser(
	ctx context.Context,
	in *minderv1.GetUserRequest,
	opts ...grpc.CallOption,
) (*minderv1.GetUserResponse, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, in, opts...)
	}
	return nil, nil
}

func TestUserRegistered(t *testing.T) {
	t.Parallel()

	expectedResp := &minderv1.GetUserResponse{
		User: &minderv1.UserRecord{
			IdentitySubject: "test-user-sub",
		},
	}

	tests := []struct {
		name         string
		getUserFunc  func(ctx context.Context, in *minderv1.GetUserRequest, opts ...grpc.CallOption) (*minderv1.GetUserResponse, error)
		wantOk       bool
		wantResponse *minderv1.GetUserResponse
		wantErr      bool
	}{
		{
			name: "user successfully retrieved",
			getUserFunc: func(_ context.Context, _ *minderv1.GetUserRequest, _ ...grpc.CallOption) (*minderv1.GetUserResponse, error) {
				return expectedResp, nil
			},
			wantOk:       true,
			wantResponse: expectedResp,
			wantErr:      false,
		},
		{
			name: "user not found returns false without error",
			getUserFunc: func(_ context.Context, _ *minderv1.GetUserRequest, _ ...grpc.CallOption) (*minderv1.GetUserResponse, error) {
				return nil, status.Error(codes.NotFound, "user not registered")
			},
			wantOk:       false,
			wantResponse: nil,
			wantErr:      false,
		},
		{
			name: "internal error returns error",
			getUserFunc: func(_ context.Context, _ *minderv1.GetUserRequest, _ ...grpc.CallOption) (*minderv1.GetUserResponse, error) {
				return nil, status.Error(codes.Internal, "database error")
			},
			wantOk:       false,
			wantResponse: nil,
			wantErr:      true,
		},
		{
			name: "generic error returns error",
			getUserFunc: func(_ context.Context, _ *minderv1.GetUserRequest, _ ...grpc.CallOption) (*minderv1.GetUserResponse, error) {
				return nil, errors.New("connection reset")
			},
			wantOk:       false,
			wantResponse: nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &mockUserServiceClient{getUserFunc: tt.getUserFunc}
			ok, res, err := userRegistered(context.Background(), client)

			if tt.wantErr {
				require.Error(t, err)
				assert.False(t, ok)
				assert.Nil(t, res)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantResponse, res)
		})
	}
}

func TestGetProjectTableRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		projects []*minderv1.ProjectRole
		want     [][]string
	}{
		{
			name:     "empty projects list",
			projects: nil,
			want:     nil,
		},
		{
			name: "single project",
			projects: []*minderv1.ProjectRole{
				{
					Project: &minderv1.Project{Name: "default", ProjectId: "proj-1"},
					Role:    &minderv1.Role{Name: "admin"},
				},
			},
			want: [][]string{
				{"Project (role: admin)", "default / proj-1"},
			},
		},
		{
			name: "multiple projects formats with indices",
			projects: []*minderv1.ProjectRole{
				{
					Project: &minderv1.Project{Name: "proj-a", ProjectId: "id-a"},
					Role:    &minderv1.Role{Name: "admin"},
				},
				{
					Project: &minderv1.Project{Name: "proj-b", ProjectId: "id-b"},
					Role:    &minderv1.Role{Name: "viewer"},
				},
			},
			want: [][]string{
				{"Project #1 (role: admin)", "proj-a / id-a"},
				{"Project #2 (role: viewer)", "proj-b / id-b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getProjectTableRows(tt.projects)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRenderNewUser(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	newUser := &minderv1.CreateUserResponse{
		IdentitySubject: "sub-12345",
		ProjectId:       "proj-uuid-1",
		ProjectName:     "primary-project",
	}

	renderNewUser("minder.example.com", newUser, &buf)
	output := buf.String()

	assert.Contains(t, output, "sub-12345")
	assert.Contains(t, output, "proj-uuid-1")
	assert.Contains(t, output, "primary-project")
	assert.Contains(t, output, "minder.example.com")
}

func TestRenderUserInfo(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	user := &minderv1.GetUserResponse{
		User: &minderv1.UserRecord{
			IdentitySubject: "user-subject-abc",
		},
		ProjectRoles: []*minderv1.ProjectRole{
			{
				Project: &minderv1.Project{Name: "my-project", ProjectId: "p-123"},
				Role:    &minderv1.Role{Name: "editor"},
			},
		},
	}

	renderUserInfo("test-server:8080", user, &buf)
	output := buf.String()

	assert.Contains(t, output, "test-server:8080")
	assert.Contains(t, output, "user-subject-abc")
	assert.Contains(t, output, "my-project / p-123")
	assert.Contains(t, output, "role: editor")
}

func TestRenderUserInfoWhoami(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	user := &minderv1.GetUserResponse{
		User: &minderv1.UserRecord{
			IdentitySubject: "whoami-user",
			CreatedAt:       timestamppb.New(now),
			UpdatedAt:       timestamppb.New(now),
		},
		ProjectRoles: []*minderv1.ProjectRole{
			{
				Project: &minderv1.Project{Name: "main-project", ProjectId: "main-id"},
				Role:    &minderv1.Role{Name: "admin"},
			},
		},
	}

	t.Run("renders table format", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		renderUserInfoWhoami("minder-host", &buf, app.Table, user)
		out := buf.String()

		assert.Contains(t, out, "Here are your details:")
		assert.Contains(t, out, "whoami-user")
		assert.Contains(t, out, "minder-host")
		assert.Contains(t, out, "main-project / main-id")
	})

	t.Run("renders json format", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		renderUserInfoWhoami("minder-host", &buf, app.JSON, user)

		var decoded minderv1.GetUserResponse
		err := protojson.Unmarshal(buf.Bytes(), &decoded)
		require.NoError(t, err)

		assert.Equal(t, "whoami-user", decoded.GetUser().GetIdentitySubject())
		require.Len(t, decoded.GetProjectRoles(), 1)
		assert.Equal(t, "main-project", decoded.GetProjectRoles()[0].GetProject().GetName())
		assert.Equal(t, "main-id", decoded.GetProjectRoles()[0].GetProject().GetProjectId())
		assert.Equal(t, "admin", decoded.GetProjectRoles()[0].GetRole().GetName())
	})

	t.Run("renders yaml format", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		renderUserInfoWhoami("minder-host", &buf, app.YAML, user)

		assert.False(t, json.Valid(buf.Bytes()), "expected YAML output, got JSON")

		var decoded map[string]any
		err := yaml.Unmarshal(buf.Bytes(), &decoded)
		require.NoError(t, err)

		userMap, ok := decoded["user"].(map[string]any)
		require.True(t, ok, "expected user map in decoded yaml")
		assert.Equal(t, "whoami-user", userMap["identitySubject"])

		projectRoles, ok := decoded["projectRoles"].([]any)
		require.True(t, ok, "expected projectRoles array in decoded yaml")
		require.Len(t, projectRoles, 1)

		firstRole, ok := projectRoles[0].(map[string]any)
		require.True(t, ok)
		projectMap, ok := firstRole["project"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "main-project", projectMap["name"])
		assert.Equal(t, "main-id", projectMap["projectId"])
		roleMap, ok := firstRole["role"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "admin", roleMap["name"])
	})
}
