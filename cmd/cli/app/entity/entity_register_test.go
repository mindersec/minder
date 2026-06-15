// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestRegisterCommand(t *testing.T) {
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
			Name: "register with comma value",
			Args: []string{
				"entity", "register",
				"--type", "repository",
				"--property", "github/repo_topics=topic1,topic2,topic3",
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				mockResp := &minderv1.ListEntitiesResponse{}
				cli.LoadFixture(t, "mock_entities_response.json", mockResp)

				// Use a Do function to verify that the property value is correctly parsed into a list
				client.EXPECT().
					RegisterEntity(gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, req *minderv1.RegisterEntityRequest, _ ...grpc.CallOption) {
						props := req.GetIdentifyingProperties()
						topics, ok := props["github/repo_topics"]
						if !ok {
							t.Fatal("missing github/repo_topics property")
						}
						require.Equal(t, "topic1,topic2,topic3", topics.GetStringValue(), "expected raw string value for comma-separated topics")
					}).
					Return(&minderv1.RegisterEntityResponse{Entity: mockResp.Results[0]}, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "register_comma_value.json",
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
