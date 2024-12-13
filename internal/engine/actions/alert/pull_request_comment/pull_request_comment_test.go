// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request_comment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/db"
	enginerr "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/interfaces"
	pbinternal "github.com/mindersec/minder/internal/proto"
	mockghclient "github.com/mindersec/minder/internal/providers/github/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles/models"
)

var TestActionTypeValid interfaces.ActionType = "alert-test"

const evaluationFailureDetails = "evaluation failure reason"

func TestPullRequestCommentAlert(t *testing.T) {
	t.Parallel()

	reviewID := int64(456)
	reviewIDStr := strconv.FormatInt(reviewID, 10)
	successfulRunMetadata := json.RawMessage(fmt.Sprintf(`{"review_id":"%s"}`, reviewIDStr))

	tests := []struct {
		name             string
		actionType       interfaces.ActionType
		cmd              interfaces.ActionCmd
		reviewMsg        string
		inputMetadata    *json.RawMessage
		mockSetup        func(*mockghclient.MockGitHub)
		expectedErr      error
		expectedMetadata json.RawMessage
	}{
		{
			name:       "create a PR comment",
			actionType: TestActionTypeValid,
			reviewMsg:  "This is a constant review message",
			cmd:        interfaces.ActionCmdOn,
			mockSetup: func(mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					CreateReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&github.PullRequestReview{ID: &reviewID}, nil)
			},
			expectedMetadata: json.RawMessage(fmt.Sprintf(`{"review_id":"%s"}`, reviewIDStr)),
		},
		{
			name:       "create a PR comment with template",
			actionType: TestActionTypeValid,
			reviewMsg:  "{{ .EvalErrorDetails }}",
			cmd:        interfaces.ActionCmdOn,
			mockSetup: func(mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					CreateReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&github.PullRequestReviewRequest{})).
					DoAndReturn(func(_ context.Context, _, _ string, _ int, review *github.PullRequestReviewRequest) (*github.PullRequestReview, error) {
						if review.GetBody() != evaluationFailureDetails {
							return nil, fmt.Errorf("expected review body to be %s, got %s", evaluationFailureDetails, review.GetBody())
						}
						return &github.PullRequestReview{ID: &reviewID}, nil
					})
			},
			expectedMetadata: json.RawMessage(fmt.Sprintf(`{"review_id":"%s"}`, reviewIDStr)),
		},
		{
			name:       "error from provider creating PR comment",
			actionType: TestActionTypeValid,
			reviewMsg:  "This is a constant review message",
			cmd:        interfaces.ActionCmdOn,
			mockSetup: func(mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					CreateReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("failed to create PR comment"))
			},
			expectedErr: enginerr.ErrActionFailed,
		},
		{
			name:          "dismiss PR comment",
			actionType:    TestActionTypeValid,
			reviewMsg:     "This is a constant review message",
			cmd:           interfaces.ActionCmdOff,
			inputMetadata: &successfulRunMetadata,
			mockSetup: func(mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					DismissReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&github.PullRequestReview{}, nil)
			},
			expectedErr: enginerr.ErrActionTurnedOff,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() {
				ctrl.Finish()
			})

			prCommentCfg := pb.RuleType_Definition_Alert_AlertTypePRComment{
				ReviewMessage: tt.reviewMsg,
			}

			mockClient := mockghclient.NewMockGitHub(ctrl)
			tt.mockSetup(mockClient)

			prCommentAlert, err := NewPullRequestCommentAlert(
				tt.actionType, &prCommentCfg, mockClient, models.ActionOptOn)
			require.NoError(t, err)
			require.NotNil(t, prCommentAlert)

			evalParams := &interfaces.EvalStatusParams{
				EvalStatusFromDb: &db.ListRuleEvaluationsByProfileIdRow{},
				Profile:          &models.ProfileAggregate{},
				Rule:             &models.RuleInstance{},
			}
			evalParams.SetEvalErr(enginerr.NewErrEvaluationFailed(evaluationFailureDetails))

			retMeta, err := prCommentAlert.Do(
				context.Background(),
				tt.cmd,
				&pbinternal.PullRequest{},
				evalParams,
				tt.inputMetadata,
			)
			require.ErrorIs(t, err, tt.expectedErr, "expected error")
			require.Equal(t, tt.expectedMetadata, retMeta)
		})
	}
}
