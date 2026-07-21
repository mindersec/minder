// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"path/filepath"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestApplyCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name: "apply - create profile and ruletypes with positional arg",
			Args: []string{"apply", "fixture"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				dsClient := mockv1.NewMockDataSourceServiceClient(ctrl)
				rtClient := mockv1.NewMockRuleTypeServiceClient(ctrl)
				pClient := mockv1.NewMockProfileServiceClient(ctrl)
				mockRTResp := &minderv1.ListRuleTypesResponse{}
				mockProfileResp := &minderv1.ListProfilesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockRTResp)
				cli.LoadFixture(t, "mock_profiles_response.json", mockProfileResp)

				rtClient.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{RuleType: mockRTResp.RuleTypes[0]}, nil)
				rtClient.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{RuleType: mockRTResp.RuleTypes[1]}, nil)
				pClient.EXPECT().
					CreateProfile(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateProfileResponse{Profile: mockProfileResp.Profiles[0]}, nil)
				ctx := context.Background()
				ctx = cli.WithRPCClient[minderv1.DataSourceServiceClient](ctx, dsClient)
				ctx = cli.WithRPCClient[minderv1.RuleTypeServiceClient](ctx, rtClient)
				ctx = cli.WithRPCClient[minderv1.ProfileServiceClient](ctx, pClient)
				return ctx
			},
			GoldenFileName: "apply_create.table",
		},
		{
			Name: "apply - create profile and ruletypes with flag",
			Args: []string{"apply", "-f", "fixture/"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				dsClient := mockv1.NewMockDataSourceServiceClient(ctrl)
				rtClient := mockv1.NewMockRuleTypeServiceClient(ctrl)
				pClient := mockv1.NewMockProfileServiceClient(ctrl)
				mockRTResp := &minderv1.ListRuleTypesResponse{}
				mockProfileResp := &minderv1.ListProfilesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockRTResp)
				cli.LoadFixture(t, "mock_profiles_response.json", mockProfileResp)

				rtClient.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{RuleType: mockRTResp.RuleTypes[0]}, nil)
				rtClient.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{RuleType: mockRTResp.RuleTypes[1]}, nil)
				pClient.EXPECT().
					CreateProfile(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateProfileResponse{Profile: mockProfileResp.Profiles[0]}, nil)
				ctx := context.Background()
				ctx = cli.WithRPCClient[minderv1.DataSourceServiceClient](ctx, dsClient)
				ctx = cli.WithRPCClient[minderv1.RuleTypeServiceClient](ctx, rtClient)
				ctx = cli.WithRPCClient[minderv1.ProfileServiceClient](ctx, pClient)
				return ctx
			},
			GoldenFileName: "apply_create.table",
		},
		{
			Name:          "no files specified",
			Args:          []string{"apply"},
			ExpectedError: "no resources found",
		},
		{
			Name:          "only empty files",
			Args:          []string{"apply", filepath.Join("fixture", "empty_file.yaml")},
			ExpectedError: "no resources found",
		},
	}

	cli.RunCmdTests(t, tests, applyCmd)
}
