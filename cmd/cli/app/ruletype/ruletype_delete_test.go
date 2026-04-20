// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestDeleteCommand(t *testing.T) {
	const (
		ruleID1 = "00000000-0000-0000-0000-000000000001"
		ruleID2 = "00000000-0000-0000-0000-000000000002"
	)

	tests := []cli.CmdTestCase{
		{
			Name: "delete single rule type by id",
			Args: []string{"ruletype", "delete", "--id", ruleID1},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)
				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				// command calls GetRuleTypeById to verify it exists and get the name
				client.EXPECT().
					GetRuleTypeById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByIdResponse{
						RuleType: mockResp.RuleTypes[0], // secret_push_protection
					}, nil)

				// command then calls DeleteRuleType
				client.EXPECT().
					DeleteRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteRuleTypeResponse{}, nil)

				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_single.txt",
		},
		{
			Name: "delete all rule types",
			Args: []string{"ruletype", "delete", "--all", "--yes"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)

				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				// command calls ListRuleTypes to find everything to delete
				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				// command loops through and deletes each one
				// since fixture has 3 rules, we expect 3 calls
				client.EXPECT().
					DeleteRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteRuleTypeResponse{}, nil).
					Times(len(mockResp.RuleTypes))

				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_all.txt",
		},
		{
			Name: "partial failure - profile reference",
			Args: []string{"ruletype", "delete", "--id", ruleID2},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)

				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					GetRuleTypeById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByIdResponse{
						RuleType: mockResp.RuleTypes[1],
					}, nil)

				client.EXPECT().
					DeleteRuleType(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.FailedPrecondition, "cannot delete: used by profiles my-security-profile"))

				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_partial_failure.txt",
		},
		{
			Name:          "missing required flags",
			Args:          []string{"ruletype", "delete"},
			ExpectedError: "at least one of the flags in the group [id name all] is required",
		},
	}

	cli.RunCmdTests(t, tests, ruleTypeCmd)
}
