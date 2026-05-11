// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
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
func TestGetCommand(t *testing.T) {
	const (
		entityID   = "00000000-0000-0000-0000-000000000001"
		entityName = "myorg/myrepo"
	)

	tests := []cli.CmdTestCase{
		{
			Name: "get by id - table output",
			Args: []string{"entity", "get", "--id", entityID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResp := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResp)

				client.EXPECT().
					GetEntityById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetEntityByIdResponse{Entity: mockResp.Results[0]}, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "get_by_id.table",
		},
		{
			Name: "get by id - table output no emoji",
			Args: []string{"entity", "get", "--id", entityID, "--emoji=false"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResp := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResp)

				client.EXPECT().
					GetEntityById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetEntityByIdResponse{Entity: mockResp.Results[0]}, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "get_by_id_no_emoji.table",
		},
		{
			Name: "get by id - json output",
			Args: []string{"entity", "get", "--id", entityID, "-o", app.JSON},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResp := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResp)

				client.EXPECT().
					GetEntityById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetEntityByIdResponse{Entity: mockResp.Results[0]}, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "get_by_id.json",
		},
		{
			Name: "get by name - yaml output",
			Args: []string{"entity", "get", "--name", entityName, "--type", "repository", "-o", app.YAML},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResp := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResp)

				client.EXPECT().
					GetEntityByName(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetEntityByNameResponse{Entity: mockResp.Results[0]}, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "get_by_name.yaml",
		},
		{
			Name:          "missing both id and name",
			Args:          []string{"entity", "get", "-o", app.JSON},
			ExpectedError: "at least one of the flags in the group [id name] is required",
		},
		{
			Name:          "giving both id and name",
			Args:          []string{"entity", "get", "--id", entityID, "--name", entityName},
			ExpectedError: "if any flags in the group [id name] are set none of the others can be",
		},
		{
			Name: "grpc error",
			Args: []string{"entity", "get", "--id", entityID, "-o", app.JSON},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				client.EXPECT().
					GetEntityById(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "entity not found"))
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			ExpectedError: "entity not found",
		},
	}

	cli.RunCmdTests(t, tests, EntityCmd)
}
