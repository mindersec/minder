// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request_comment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	github "github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/db"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	pbinternal "github.com/mindersec/minder/internal/proto"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	enginerr "github.com/mindersec/minder/pkg/engine/errors"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles/models"
	mock_provifv1 "github.com/mindersec/minder/pkg/providers/v1/mock"
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
		name          string
		actionType    engif.ActionType
		cmd           engif.ActionCmd
		reviewMsg     string
		inputMetadata *json.RawMessage
		mockSetup     func(*mock_provifv1.MockReviewPublisher)
		expectedErr   error
		// expectMeta indicates whether we expect non-nil metadata back
		expectMeta bool
	}{
		{
			name:       "create a PR comment",
			actionType: TestActionTypeValid,
			reviewMsg:  "This is a constant review message",
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockPublisher *mock_provifv1.MockReviewPublisher) {
				mockPublisher.EXPECT().
					ListReviews(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
				mockPublisher.EXPECT().
					CreateReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&github.PullRequestReview{ID: github.Int64(reviewID)}, nil)
			},
			expectMeta: true,
		},
		{
			name:       "create a PR comment with eval error details template",
			actionType: TestActionTypeValid,
			reviewMsg:  "{{ .EvalErrorDetails }}",
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockPublisher *mock_provifv1.MockReviewPublisher) {
				mockPublisher.EXPECT().
					ListReviews(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
				mockPublisher.EXPECT().
					CreateReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _ string, _ int, req *github.PullRequestReviewRequest) (*github.PullRequestReview, error) {
						expectedBody := fmt.Sprintf("%s\n\n<!-- minder-rule: test-rule -->", evaluationFailureDetails)
						if req.GetBody() != expectedBody {
							return nil, fmt.Errorf("expected review body to be %s, got %s", expectedBody, req.GetBody())
						}
						return &github.PullRequestReview{ID: github.Int64(reviewID)}, nil
					})
			},
			expectMeta: true,
		},
		{
			name:       "create a PR comment with eval result output template",
			actionType: TestActionTypeValid,
			reviewMsg:  "{{ .EvalResultOutput.ViolationMsg }}",
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockPublisher *mock_provifv1.MockReviewPublisher) {
				mockPublisher.EXPECT().
					ListReviews(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
				mockPublisher.EXPECT().
					CreateReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _ string, _ int, req *github.PullRequestReviewRequest) (*github.PullRequestReview, error) {
						expectedBody := fmt.Sprintf("%s\n\n<!-- minder-rule: test-rule -->", violationMsg)
						if req.GetBody() != expectedBody {
							return nil, fmt.Errorf("expected review body to be %s, got %s", expectedBody, req.GetBody())
						}
						return &github.PullRequestReview{ID: github.Int64(reviewID)}, nil
					})
			},
			expectMeta: true,
		},
		{
			name:       "error from provider creating PR comment",
			actionType: TestActionTypeValid,
			reviewMsg:  "This is a constant review message",
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockPublisher *mock_provifv1.MockReviewPublisher) {
				mockPublisher.EXPECT().
					ListReviews(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
				mockPublisher.EXPECT().
					CreateReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("failed to create PR comment"))
			},
			expectedErr: enginerr.ErrActionFailed,
			expectMeta:  false,
		},
		{
			name:          "dismiss PR comment",
			actionType:    TestActionTypeValid,
			reviewMsg:     "This is a constant review message",
			cmd:           engif.ActionCmdOff,
			inputMetadata: &successfulRunMetadata,
			mockSetup: func(mockPublisher *mock_provifv1.MockReviewPublisher) {
				mockPublisher.EXPECT().
					ListReviews(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*github.PullRequestReview{
						{
							ID:   github.Int64(reviewID),
							Body: github.String("<!-- minder-rule: test-rule -->"),
						},
					}, nil)
				mockPublisher.EXPECT().
					DismissReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), reviewID, gomock.Any()).
					Return(&github.PullRequestReview{}, nil)
			},
			expectedErr: enginerr.ErrActionTurnedOff,
			expectMeta:  false,
		},
		{
			name:       "update an existing PR review",
			actionType: TestActionTypeValid,
			reviewMsg:  "Updated review message",
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockPublisher *mock_provifv1.MockReviewPublisher) {
				mockPublisher.EXPECT().
					ListReviews(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*github.PullRequestReview{
						{
							ID:   github.Int64(reviewID),
							Body: github.String("<!-- minder-rule: test-rule -->"),
						},
					}, nil)
				mockPublisher.EXPECT().
					UpdateReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), reviewID, gomock.Any()).
					Return(&github.PullRequestReview{ID: github.Int64(reviewID)}, nil)
			},
			expectMeta: true,
		},
		{
			name:       "create a PR comment with REQUEST_CHANGES",
			actionType: TestActionTypeValid,
			reviewMsg:  "Please fix this",
			cmd:        engif.ActionCmdOn,
			mockSetup: func(mockPublisher *mock_provifv1.MockReviewPublisher) {
				mockPublisher.EXPECT().
					ListReviews(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil)
				mockPublisher.EXPECT().
					CreateReview(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _ string, _ int, req *github.PullRequestReviewRequest) (*github.PullRequestReview, error) {
						if req.GetEvent() != "REQUEST_CHANGES" {
							return nil, fmt.Errorf("expected event to be REQUEST_CHANGES, got %s", req.GetEvent())
						}
						return &github.PullRequestReview{ID: github.Int64(reviewID)}, nil
					})
			},
			expectMeta: true,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() {
				ctrl.Finish()
			})

			reviewAction := ""
			if tt.name == "create a PR comment with REQUEST_CHANGES" {
				reviewAction = "request_changes"
			}

			prCommentCfg := pb.RuleType_Definition_Alert_AlertTypePRComment{
				ReviewMessage: tt.reviewMsg,
				Action:        &reviewAction,
			}

			mockClient := mock_provifv1.NewMockReviewPublisher(ctrl)
			tt.mockSetup(mockClient)

			prCommentAlert, err := NewPullRequestCommentAlert(
				tt.actionType, &prCommentCfg, mockClient, models.ActionOptOn)
			require.NoError(t, err)
			require.NotNil(t, prCommentAlert)

			evalParams := &engif.EvalStatusParams{
				EvalStatusFromDb: &db.ListRuleEvaluationsByProfileIdRow{},
				Profile:          &models.ProfileAggregate{},
				Rule:             &models.RuleInstance{Name: "test-rule"},
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
			if tt.expectMeta {
				require.NotNil(t, retMeta)
			} else {
				require.Nil(t, retMeta)
			}
		})
	}
}

type exampleOutput struct {
	ViolationMsg string
}
