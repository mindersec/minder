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
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	pbinternal "github.com/mindersec/minder/internal/proto"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles/models"
	mockcommenter "github.com/mindersec/minder/pkg/providers/v1/mock"
)

var TestActionTypeValid engif.ActionType = "alert-test"

const (
	evaluationFailureDetails = "evaluation failure reason"
	violationMsg             = "violation message"
)

func TestPullRequestCommentAlert(t *testing.T) {
	t.Parallel()

	reviewID := int64(456)
	reviewIDStr := strconv.FormatInt(reviewID, 10)
	successfulRunMetadata := json.RawMessage(fmt.Sprintf(`{"review_id":"%s"}`, reviewIDStr))

	tests := []struct {
		name             string
		actionType       engif.ActionType
		cmd              engif.ActionCmd
		reviewMsg        string
		inputMetadata    *json.RawMessage
		mockSetup        func(commenter *mockcommenter.MockPullRequestCommenter)
		expectedErr      error
		expectedMetadata json.RawMessage
	}{
		{
			name:       "create a PR comment",
			actionType: TestActionTypeValid,
			reviewMsg:  "This is a constant review message",
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockGitHub *mockcommenter.MockPullRequestCommenter) {
				mockGitHub.EXPECT().
					CommentOnPullRequest(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&github.PullRequestReview{ID: &reviewID}, nil)
			},
			expectedMetadata: json.RawMessage(fmt.Sprintf(`{"review_id":"%s"}`, reviewIDStr)),
		},
		{
			name:       "create a PR comment with eval error details template",
			actionType: TestActionTypeValid,
			reviewMsg:  "{{ .EvalErrorDetails }}",
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockGitHub *mockcommenter.MockPullRequestCommenter) {
				mockGitHub.EXPECT().
					CommentOnPullRequest(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(validateReviewBodyAndReturn(evaluationFailureDetails, reviewID))
			},
			expectedMetadata: json.RawMessage(fmt.Sprintf(`{"review_id":"%s"}`, reviewIDStr)),
		},
		{
			name:       "create a PR comment with eval result output template",
			actionType: TestActionTypeValid,
			reviewMsg:  "{{ .EvalResultOutput.ViolationMsg }}",
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockGitHub *mockcommenter.MockPullRequestCommenter) {
				mockGitHub.EXPECT().
					CommentOnPullRequest(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(validateReviewBodyAndReturn(violationMsg, reviewID))
			},
			expectedMetadata: json.RawMessage(fmt.Sprintf(`{"review_id":"%s"}`, reviewIDStr)),
		},
		{
			name:       "error from provider creating PR comment",
			actionType: TestActionTypeValid,
			reviewMsg:  "This is a constant review message",
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockGitHub *mockcommenter.MockPullRequestCommenter) {
				mockGitHub.EXPECT().
					CommentOnPullRequest(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("failed to create PR comment"))
			},
			expectedErr: enginerr.ErrActionFailed,
		},
		{
			name:          "dismiss PR comment",
			actionType:    TestActionTypeValid,
			reviewMsg:     "This is a constant review message",
			cmd:           engif.ActionCmdOff,
			inputMetadata: &successfulRunMetadata,
			mockSetup: func(mockGitHub *mockcommenter.MockPullRequestCommenter) {
				mockGitHub.EXPECT().
					CommentOnPullRequest(gomock.Any(), gomock.Any(), gomock.Any()).
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

			mockClient := mockcommenter.NewMockPullRequestCommenter(ctrl)
			tt.mockSetup(mockClient)

			prCommentAlert, err := NewPullRequestCommentAlert(
				tt.actionType, &prCommentCfg, mockClient, models.ActionOptOn, "Title")
			require.NoError(t, err)
			require.NotNil(t, prCommentAlert)

			evalParams := &engif.EvalStatusParams{
				EvalStatusFromDb: &db.ListRuleEvaluationsByProfileIdRow{},
				Profile:          &models.ProfileAggregate{},
				Rule:             &models.RuleInstance{},
			}
			evalParams.SetEvalErr(enginerr.NewErrEvaluationFailed(evaluationFailureDetails))
			evalParams.SetEvalResult(&interfaces.EvaluationResult{
				Output: exampleOutput{ViolationMsg: violationMsg},
			})

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

type exampleOutput struct {
	ViolationMsg string
}

func validateReviewBodyAndReturn(expectedBody string, reviewID int64) func(_ context.Context, _, _ string, _ int, review *github.PullRequestReviewRequest) (*github.PullRequestReview, error) {
	return func(_ context.Context, _, _ string, _ int, review *github.PullRequestReviewRequest) (*github.PullRequestReview, error) {
		if review.GetBody() != expectedBody {
			return nil, fmt.Errorf("expected review body to be %s, got %s", expectedBody, review.GetBody())
		}
		return &github.PullRequestReview{ID: &reviewID}, nil
	}
}
