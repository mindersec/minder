// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

type artifactClientStub struct {
	getArtifactByIDResp   *minderv1.GetArtifactByIdResponse
	getArtifactByIDErr    error
	getArtifactByNameResp *minderv1.GetArtifactByNameResponse
	getArtifactByNameErr  error
}

func (s *artifactClientStub) ListArtifacts(context.Context, *minderv1.ListArtifactsRequest, ...grpc.CallOption) (*minderv1.ListArtifactsResponse, error) {
	_ = s
	return nil, status.Error(codes.Unimplemented, "unexpected call")
}

func (s *artifactClientStub) GetArtifactById(_ context.Context, _ *minderv1.GetArtifactByIdRequest, _ ...grpc.CallOption) (*minderv1.GetArtifactByIdResponse, error) {
	return s.getArtifactByIDResp, s.getArtifactByIDErr
}

func (s *artifactClientStub) GetArtifactByName(_ context.Context, _ *minderv1.GetArtifactByNameRequest, _ ...grpc.CallOption) (*minderv1.GetArtifactByNameResponse, error) {
	return s.getArtifactByNameResp, s.getArtifactByNameErr
}

type profileClientStub struct {
	listProfilesResp *minderv1.ListProfilesResponse
	listProfilesErr  error
	statusResp       *minderv1.GetProfileStatusByNameResponse
	statusErr        error
}

func (s *profileClientStub) CreateProfile(context.Context, *minderv1.CreateProfileRequest, ...grpc.CallOption) (*minderv1.CreateProfileResponse, error) {
	_ = s
	return nil, status.Error(codes.Unimplemented, "unexpected call")
}

func (s *profileClientStub) UpdateProfile(context.Context, *minderv1.UpdateProfileRequest, ...grpc.CallOption) (*minderv1.UpdateProfileResponse, error) {
	_ = s
	return nil, status.Error(codes.Unimplemented, "unexpected call")
}

func (s *profileClientStub) PatchProfile(context.Context, *minderv1.PatchProfileRequest, ...grpc.CallOption) (*minderv1.PatchProfileResponse, error) {
	_ = s
	return nil, status.Error(codes.Unimplemented, "unexpected call")
}

func (s *profileClientStub) DeleteProfile(context.Context, *minderv1.DeleteProfileRequest, ...grpc.CallOption) (*minderv1.DeleteProfileResponse, error) {
	_ = s
	return nil, status.Error(codes.Unimplemented, "unexpected call")
}

func (s *profileClientStub) ListProfiles(_ context.Context, _ *minderv1.ListProfilesRequest, _ ...grpc.CallOption) (*minderv1.ListProfilesResponse, error) {
	return s.listProfilesResp, s.listProfilesErr
}

func (s *profileClientStub) GetProfileById(context.Context, *minderv1.GetProfileByIdRequest, ...grpc.CallOption) (*minderv1.GetProfileByIdResponse, error) {
	_ = s
	return nil, status.Error(codes.Unimplemented, "unexpected call")
}

func (s *profileClientStub) GetProfileByName(context.Context, *minderv1.GetProfileByNameRequest, ...grpc.CallOption) (*minderv1.GetProfileByNameResponse, error) {
	_ = s
	return nil, status.Error(codes.Unimplemented, "unexpected call")
}

func (s *profileClientStub) GetProfileStatusByName(_ context.Context, _ *minderv1.GetProfileStatusByNameRequest, _ ...grpc.CallOption) (*minderv1.GetProfileStatusByNameResponse, error) {
	return s.statusResp, s.statusErr
}

func (s *profileClientStub) GetProfileStatusById(context.Context, *minderv1.GetProfileStatusByIdRequest, ...grpc.CallOption) (*minderv1.GetProfileStatusByIdResponse, error) {
	_ = s
	return nil, status.Error(codes.Unimplemented, "unexpected call")
}

func (s *profileClientStub) GetProfileStatusByProject(context.Context, *minderv1.GetProfileStatusByProjectRequest, ...grpc.CallOption) (*minderv1.GetProfileStatusByProjectResponse, error) {
	_ = s
	return nil, status.Error(codes.Unimplemented, "unexpected call")
}

// Tests rely on injected ArtifactServiceClient; profile evaluation is skipped in this mode.
//
//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestArtifactGetCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, _ *gomock.Controller) context.Context {
		t.Helper()
		client := &artifactClientStub{}

		mockResp := &minderv1.GetArtifactByIdResponse{}
		cli.LoadFixture(t, "mock_artifact_get.json", mockResp)

		client.getArtifactByIDResp = mockResp
		client.getArtifactByIDErr = nil

		profileClient := &profileClientStub{
			listProfilesResp: &minderv1.ListProfilesResponse{},
			statusResp:       &minderv1.GetProfileStatusByNameResponse{},
		}

		ctx := cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), client)
		ctx = cli.WithRPCClient[minderv1.ProfileServiceClient](ctx, profileClient)
		return ctx
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "get artifact - table output",
			Args:           []string{"artifact", "get", "-i", "111", "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_get.table",
		},
		{
			Name:           "get artifact - json output",
			Args:           []string{"artifact", "get", "-i", "111", "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_get.json",
		},
		{
			Name:           "get artifact - yaml output",
			Args:           []string{"artifact", "get", "-i", "111", "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_get.yaml",
		},
		{
			Name: "server error handling",
			Args: []string{"artifact", "get", "-i", "111"},
			MockSetup: func(t *testing.T, _ *gomock.Controller) context.Context {
				t.Helper()
				client := &artifactClientStub{getArtifactByIDErr: status.Error(codes.NotFound, "artifact not found")}
				profileClient := &profileClientStub{listProfilesResp: &minderv1.ListProfilesResponse{}, statusResp: &minderv1.GetProfileStatusByNameResponse{}}

				ctx := cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), client)
				ctx = cli.WithRPCClient[minderv1.ProfileServiceClient](ctx, profileClient)
				return ctx
			},
			ExpectedError: "artifact not found",
		},
	}

	cli.RunCmdTests(t, tests, ArtifactCmd)
}
