// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"context"
	"os"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestExportCommand(t *testing.T) {
	testID := "00000000-0000-0000-0000-000000000001"
	testName := "mock-profile"
	exportTmpFile := "test-export-output.yaml"

	tests := []cli.CmdTestCase{
		{
			Name: "export by name success with resources",
			Args: []string{"profile", "export", "-n", testName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				profClient, rtClient, dsClient := setupAllMocks(t, ctrl)

				mockProf := &minderv1.Profile{}
				cli.LoadFixture(t, "mock_profile_export.json", mockProf)
				profClient.EXPECT().
					GetProfileByName(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetProfileByNameResponse{Profile: mockProf}, nil)

				mockRT := &minderv1.RuleType{}
				cli.LoadFixture(t, "mock_ruletype_export.json", mockRT)
				rtClient.EXPECT().
					GetRuleTypeByName(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByNameResponse{RuleType: mockRT}, nil)

				mockDS := &minderv1.DataSource{}
				cli.LoadFixture(t, "mock_datasource_export.json", mockDS)
				dsClient.EXPECT().
					GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetDataSourceByNameResponse{DataSource: mockDS}, nil)

				return setupExportContext(profClient, rtClient, dsClient)
			},
			GoldenFileName: "export_by_name.yaml",
		},
		{
			Name: "export by id success",
			Args: []string{"profile", "export", "--id", testID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				profClient, rtClient, dsClient := setupAllMocks(t, ctrl)

				mockProf := &minderv1.Profile{}
				cli.LoadFixture(t, "mock_profile_export.json", mockProf)
				profClient.EXPECT().
					GetProfileById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetProfileByIdResponse{Profile: mockProf}, nil)

				mockRT := &minderv1.RuleType{}
				cli.LoadFixture(t, "mock_ruletype_export.json", mockRT)
				rtClient.EXPECT().GetRuleTypeByName(gomock.Any(), gomock.Any()).Return(&minderv1.GetRuleTypeByNameResponse{RuleType: mockRT}, nil)

				mockDS := &minderv1.DataSource{}
				cli.LoadFixture(t, "mock_datasource_export.json", mockDS)
				dsClient.EXPECT().GetDataSourceByName(gomock.Any(), gomock.Any()).Return(&minderv1.GetDataSourceByNameResponse{DataSource: mockDS}, nil)

				return setupExportContext(profClient, rtClient, dsClient)
			},
			GoldenFileName: "export_by_id.yaml",
		},
		{
			Name: "export to file success",
			Args: []string{"profile", "export", "-n", testName, "-o", exportTmpFile},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				profClient, rtClient, dsClient := setupAllMocks(t, ctrl)

				mockProf := &minderv1.Profile{}
				profClient.EXPECT().GetProfileByName(gomock.Any(), gomock.Any()).Return(&minderv1.GetProfileByNameResponse{Profile: mockProf}, nil)

				return setupExportContext(profClient, rtClient, dsClient)
			},
			GoldenFileName: "export_to_file.txt",
		},
		{
			Name:          "failure no id or name provided",
			Args:          []string{"profile", "export"},
			ExpectedError: "id or name required",
		},
		{
			Name: "failure profile not found",
			Args: []string{"profile", "export", "-n", "missing-profile"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				profClient, rtClient, dsClient := setupAllMocks(t, ctrl)

				profClient.EXPECT().
					GetProfileByName(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "profile not found"))

				return setupExportContext(profClient, rtClient, dsClient)
			},
			ExpectedError: "profile not found",
		},
		{
			Name: "failure rule type fetch error",
			Args: []string{"profile", "export", "-n", testName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				profClient, rtClient, dsClient := setupAllMocks(t, ctrl)

				mockProf := &minderv1.Profile{}
				cli.LoadFixture(t, "mock_profile_export.json", mockProf)
				profClient.EXPECT().GetProfileByName(gomock.Any(), gomock.Any()).Return(&minderv1.GetProfileByNameResponse{Profile: mockProf}, nil)

				rtClient.EXPECT().
					GetRuleTypeByName(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "database error"))

				return setupExportContext(profClient, rtClient, dsClient)
			},
			ExpectedError: "database error",
		},
	}

	cli.RunCmdTests(t, tests, ProfileCmd)
	_ = os.Remove(exportTmpFile)
}

func setupAllMocks(t *testing.T, ctrl *gomock.Controller) (*mockv1.MockProfileServiceClient, *mockv1.MockRuleTypeServiceClient, *mockv1.MockDataSourceServiceClient) {
	t.Helper()
	return mockv1.NewMockProfileServiceClient(ctrl),
		mockv1.NewMockRuleTypeServiceClient(ctrl),
		mockv1.NewMockDataSourceServiceClient(ctrl)
}

func setupExportContext(p *mockv1.MockProfileServiceClient, r *mockv1.MockRuleTypeServiceClient, d *mockv1.MockDataSourceServiceClient) context.Context {
	ctx := context.Background()
	ctx = cli.WithRPCClient[minderv1.ProfileServiceClient](ctx, p)
	ctx = cli.WithRPCClient[minderv1.RuleTypeServiceClient](ctx, r)
	ctx = cli.WithRPCClient[minderv1.DataSourceServiceClient](ctx, d)
	return ctx
}
