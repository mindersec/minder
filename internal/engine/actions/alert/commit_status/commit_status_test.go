// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package commit_status

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	enginerr "github.com/mindersec/minder/internal/engine/errors"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	pbinternal "github.com/mindersec/minder/internal/proto"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles/models"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
	mock_provifv1 "github.com/mindersec/minder/pkg/providers/v1/mock"
)

var TestActionTypeValid engif.ActionType = "alert-test"

func TestCommitStatusAlert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		actionType    engif.ActionType
		cmd           engif.ActionCmd
		inputMetadata *json.RawMessage
		mockSetup     func(*mock_provifv1.MockCommitStatusPublisher)
		expectedErr   error
		expectMeta    bool
	}{
		{
			name:       "set commit status failure on alert On",
			actionType: TestActionTypeValid,
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockPublisher *mock_provifv1.MockCommitStatusPublisher) {
				mockPublisher.EXPECT().
					PublishCommitStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, status *provifv1.CommitStatus) error {
						if status.State != provifv1.CommitStatusFailure {
							return fmt.Errorf("expected failure state, got %s", status.State)
						}
						return nil
					})
			},
			expectMeta: true,
		},
		{
			name:       "set commit status success on alert Off",
			actionType: TestActionTypeValid,
			cmd:        engif.ActionCmdOff,
			mockSetup: func(mockPublisher *mock_provifv1.MockCommitStatusPublisher) {
				mockPublisher.EXPECT().
					PublishCommitStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _, _ string, status *provifv1.CommitStatus) error {
						if status.State != provifv1.CommitStatusSuccess {
							return fmt.Errorf("expected success state, got %s", status.State)
						}
						return nil
					})
			},
			expectedErr: enginerr.ErrActionTurnedOff,
			expectMeta:  false,
		},
		{
			name:       "error from provider setting commit status",
			actionType: TestActionTypeValid,
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockPublisher *mock_provifv1.MockCommitStatusPublisher) {
				mockPublisher.EXPECT().
					PublishCommitStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("failed to set commit status"))
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

			mockClient := mock_provifv1.NewMockCommitStatusPublisher(ctrl)
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

			pr := &pbinternal.PullRequest{
				RepoOwner: "test-owner",
				RepoName:  "test-repo",
				CommitSha: "test-sha",
				Number:    1,
			}

			var prevStatusMeta *json.RawMessage
			if tt.inputMetadata != nil {
				prevStatusMeta = tt.inputMetadata
			}

			meta, err := statusAlert.Do(context.Background(), tt.cmd, pr, evalParams, prevStatusMeta)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}

			if tt.expectMeta {
				require.NotNil(t, meta)
			}
		})
	}
}
