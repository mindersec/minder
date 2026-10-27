// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

// validOpenIDJWT is an unsigned JWT containing sub, given_name, family_name, email
const validOpenIDJWT = "eyJhbGciOiAibm9uZSIsICJ0eXAiOiAiSldUIn0." +
	"eyJzdWIiOiAidGVzdC1zdWIiLCAiZ2l2ZW5fbmFtZSI6ICJBbGljZSIsICJmYW1pbHlfbmFtZSI6ICJTbWl0aCIsICJlbWFpbCI6ICJhbGljZUBleGFtcGxlLmNvbSJ9."

func setupDeleteTestCredentials(t *testing.T, token string) {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	minderDir := filepath.Join(tempDir, "minder")
	err := os.MkdirAll(minderDir, 0o700)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(minderDir, "config.yaml"), []byte{}, 0o600)
	require.NoError(t, err)

	creds := cli.OpenIdCredentials{
		AccessToken:          token,
		RefreshToken:         "test-refresh-token",
		AccessTokenExpiresAt: time.Now().Add(24 * time.Hour),
	}
	credsData, err := json.Marshal(creds)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(minderDir, "credentials.json"), credsData, 0o600)
	require.NoError(t, err)
}

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout/Env state
func TestDeleteCommand(t *testing.T) {
	t.Run("successfully deletes user account with yes flag", func(t *testing.T) {
		setupDeleteTestCredentials(t, validOpenIDJWT)

		tests := []cli.CmdTestCase{
			{
				Name: "deletes account when confirmed via flag",
				Args: []string{"auth", "delete", "--yes-delete-my-account"},
				MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
					t.Helper()
					client := mockv1.NewMockUserServiceClient(ctrl)

					client.EXPECT().
						GetUser(gomock.Any(), gomock.Any()).
						Return(&minderv1.GetUserResponse{}, nil)

					client.EXPECT().
						DeleteUser(gomock.Any(), gomock.Any()).
						Return(&minderv1.DeleteUserResponse{}, nil)

					return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), client)
				},
				ExpectEmptyStdout: true,
				ExpectedStderr:    "Successfully deleted account.",
			},
		}

		cli.RunCmdTests(t, tests, AuthCmd)
	})

	t.Run("fails when user does not exist", func(t *testing.T) {
		setupDeleteTestCredentials(t, validOpenIDJWT)

		tests := []cli.CmdTestCase{
			{
				Name: "fails if user is not found in database",
				Args: []string{"auth", "delete", "--yes-delete-my-account"},
				MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
					t.Helper()
					client := mockv1.NewMockUserServiceClient(ctrl)

					client.EXPECT().
						GetUser(gomock.Any(), gomock.Any()).
						Return(nil, status.Error(codes.NotFound, "user not found"))

					return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), client)
				},
				ExpectedError: "user does not exist",
			},
		}

		cli.RunCmdTests(t, tests, AuthCmd)
	})

	t.Run("fails on server error during DeleteUser RPC", func(t *testing.T) {
		setupDeleteTestCredentials(t, validOpenIDJWT)

		tests := []cli.CmdTestCase{
			{
				Name: "server error propagating during deletion",
				Args: []string{"auth", "delete", "--yes-delete-my-account"},
				MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
					t.Helper()
					client := mockv1.NewMockUserServiceClient(ctrl)

					client.EXPECT().
						GetUser(gomock.Any(), gomock.Any()).
						Return(&minderv1.GetUserResponse{}, nil)

					client.EXPECT().
						DeleteUser(gomock.Any(), gomock.Any()).
						Return(nil, status.Error(codes.Internal, "database error"))

					return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), client)
				},
				ExpectedError: "database error",
			},
		}

		cli.RunCmdTests(t, tests, AuthCmd)
	})

	t.Run("fails on missing or corrupt credentials", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		tests := []cli.CmdTestCase{
			{
				Name: "fails when credentials file is missing",
				Args: []string{"auth", "delete", "--yes-delete-my-account"},
				MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
					t.Helper()
					client := mockv1.NewMockUserServiceClient(ctrl)

					client.EXPECT().
						GetUser(gomock.Any(), gomock.Any()).
						Return(&minderv1.GetUserResponse{}, nil)

					return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), client)
				},
				ExpectedError: "error reading credentials file",
			},
		}

		cli.RunCmdTests(t, tests, AuthCmd)
	})

	t.Run("fails on malformed token string in credentials", func(t *testing.T) {
		setupDeleteTestCredentials(t, "not-a-valid-jwt")

		tests := []cli.CmdTestCase{
			{
				Name: "fails parsing non-jwt token",
				Args: []string{"auth", "delete", "--yes-delete-my-account"},
				MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
					t.Helper()
					client := mockv1.NewMockUserServiceClient(ctrl)

					client.EXPECT().
						GetUser(gomock.Any(), gomock.Any()).
						Return(&minderv1.GetUserResponse{}, nil)

					return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), client)
				},
				ExpectedError: "failed to parse string",
			},
		}

		cli.RunCmdTests(t, tests, AuthCmd)
	})
}
