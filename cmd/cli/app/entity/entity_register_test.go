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
func TestRegisterCommand(t *testing.T) {
	const entityID = "00000000-0000-0000-0000-000000000003"

	tests := []cli.CmdTestCase{
		{
			Name: "register with properties - table output",
			Args: []string{
				"entity", "register",
				"--type", "repository",
				"--property", "github/repo_owner=myorg",
				"--property", "github/repo_name=myrepo",
				"-o", app.Table,
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResp := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResp)

				client.EXPECT().
					RegisterEntity(gomock.Any(), gomock.Any()).
					Return(&minderv1.RegisterEntityResponse{Entity: mockResp.Results[0]}, nil)

				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "register_success.txt",
		},
		{
			Name: "register with properties - json output",
			Args: []string{
				"entity", "register",
				"--type", "repository",
				"--property", "github/repo_owner=myorg",
				"--property", "github/repo_name=myrepo",
				"-o", app.JSON,
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResp := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResp)

				client.EXPECT().
					RegisterEntity(gomock.Any(), gomock.Any()).
					Return(&minderv1.RegisterEntityResponse{Entity: mockResp.Results[0]}, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "register_success.json",
		},
		{
			Name:          "missing required type flag",
			Args:          []string{"entity", "register"},
			ExpectedError: "required flag(s) \"type\" not set",
		},
		{
			Name:          "invalid entity type",
			Args:          []string{"entity", "register", "--type", "foobar"},
			ExpectedError: "invalid or unspecified entity type",
		},
		{
			Name: "invalid property format",
			Args: []string{"entity", "register", "--type", "repository", "--property", "noequalssign"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			ExpectedError: "invalid property",
		},
		{
			Name: "grpc error",
			Args: []string{
				"entity", "register",
				"--type", "repository",
				"--property", "github/repo_owner=myorg",
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				client.EXPECT().
					RegisterEntity(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.InvalidArgument, "provider not found"))
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			ExpectedError: "provider not found",
		},
	}

	cli.RunCmdTests(t, tests, EntityCmd)
}
