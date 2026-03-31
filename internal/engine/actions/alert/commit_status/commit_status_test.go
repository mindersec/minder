// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package commit_status

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/db"
	enginerr "github.com/mindersec/minder/internal/engine/errors"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	pbinternal "github.com/mindersec/minder/internal/proto"
	mockghclient "github.com/mindersec/minder/internal/providers/github/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles/models"
)

var TestActionTypeValid engif.ActionType = "alert-test"

func TestCommitStatusAlert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		actionType    engif.ActionType
		cmd           engif.ActionCmd
		inputMetadata *json.RawMessage
		mockSetup     func(*mockghclient.MockGitHub)
		expectedErr   error
		expectMeta    bool
	}{
		{
			name:       "set commit status failure on alert On",
			actionType: TestActionTypeValid,
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					SetCommitStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, status *github.RepoStatus) (*github.RepoStatus, error) {
						if status.GetState() != "failure" {
							return nil, fmt.Errorf("expected failure state, got %s", status.GetState())
						}
						return status, nil
					})
			},
			expectMeta: true,
		},
		{
			name:       "set commit status success on alert Off",
			actionType: TestActionTypeValid,
			cmd:        engif.ActionCmdOff,
			mockSetup: func(mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					SetCommitStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, status *github.RepoStatus) (*github.RepoStatus, error) {
						if status.GetState() != "success" {
							return nil, fmt.Errorf("expected success state, got %s", status.GetState())
						}
						return status, nil
					})
			},
			expectedErr: enginerr.ErrActionTurnedOff,
			expectMeta:  false,
		},
		{
			name:       "error from provider setting commit status",
			actionType: TestActionTypeValid,
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					SetCommitStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("failed to set commit status"))
			},
			expectedErr: enginerr.ErrActionFailed,
			expectMeta:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() {
				ctrl.Finish()
			})

			statusCfg := pb.RuleType_Definition_Alert_AlertTypeCommitStatus{}

			mockClient := mockghclient.NewMockGitHub(ctrl)
			tt.mockSetup(mockClient)

			statusAlert, err := NewCommitStatusAlert(
				tt.actionType, &statusCfg, mockClient, models.ActionOptOn)
			require.NoError(t, err)
			require.NotNil(t, statusAlert)

			evalParams := &engif.EvalStatusParams{
				Rule: &models.RuleInstance{
					Name: "test_rule",
				},
			}

			if tt.inputMetadata != nil {
				evalParams.EvalStatusFromDb = &db.ListRuleEvaluationsByProfileIdRow{
					AlertMetadata: *tt.inputMetadata,
				}
			}

			retMeta, err := statusAlert.Do(
				context.Background(),
				tt.cmd,
				&pbinternal.PullRequest{},
				evalParams,
				tt.inputMetadata,
			)
			require.ErrorIs(t, err, tt.expectedErr, "expected error type mismatch")

			if tt.expectMeta {
				require.NotNil(t, retMeta, "expected metadata to be returned")
			} else {
				require.Nil(t, retMeta, "expected metadata to be nil")
			}
		})
	}
}
