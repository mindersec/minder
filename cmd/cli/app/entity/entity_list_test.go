// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestListCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name: "table output with data",
			Args: []string{"entity", "list", "--type", "repository", "-o", app.Table},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResponse := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResponse)

				client.EXPECT().
					ListEntities(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_populated.table",
		},
		{
			Name: "table output with data no emoji",
			Args: []string{"entity", "list", "--type", "repository", "-o", app.Table, "--emoji=false"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResponse := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResponse)

				client.EXPECT().
					ListEntities(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_populated_no_emoji.table",
		},
		{
			Name: "table output empty",
			Args: []string{"entity", "list", "--type", "repository", "-o", app.Table},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				client.EXPECT().
					ListEntities(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListEntitiesResponse{
						Results: []*minderv1.EntityInstance{},
					}, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_empty.table",
		},
		{
			Name: "json output",
			Args: []string{"entity", "list", "--type", "repository", "-o", app.JSON},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResponse := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResponse)

				client.EXPECT().
					ListEntities(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_populated.json",
		},
		{
			Name: "yaml output",
			Args: []string{"entity", "list", "--type", "repository", "-o", app.YAML},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResponse := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResponse)

				client.EXPECT().
					ListEntities(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_populated.yaml",
		},
		{
			Name: "table output with properties",
			Args: []string{"entity", "list", "--type", "repository", "--property", "github/repo_owner", "--property", "is_private"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResponse := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResponse)

				client.EXPECT().
					ListEntities(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_with_properties.table",
		},
		{
			Name:          "invalid entity type",
			Args:          []string{"entity", "list", "--type", "foobar"},
			ExpectedError: "invalid or unspecified entity type",
		},
		{
			Name:          "missing required type flag",
			Args:          []string{"entity", "list"},
			ExpectedError: "required flag(s) \"type\" not set",
		},
		{
			Name: "grpc error handling",
			Args: []string{"entity", "list", "--type", "repository", "-o", app.Table},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				client.EXPECT().
					ListEntities(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.DeadlineExceeded, "request timed out"))
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			ExpectedError: "request timed out",
		},
	}

	cli.RunCmdTests(t, tests, EntityCmd)
}
