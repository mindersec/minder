// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package issue provides the issue remediation engine.
package issue

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/engine/interfaces"
	mockghclient "github.com/mindersec/minder/internal/providers/github/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/errors"
	interfaces2 "github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles/models"
)

const (
	repoOwner = "stacklok"
	repoName  = "minder"

	issueTitle = "Dependency vulnerability detected"

	issueBody = "A dependency vulnerability has been detected."
)

var TestActionTypeValid interfaces.ActionType = "remediate-test"

func defaultIssueRem() *pb.RuleType_Definition_Remediate_IssueRemediation {
	return &pb.RuleType_Definition_Remediate_IssueRemediation{
		Title: issueTitle,
		Body:  issueBody,
	}
}

type remediateArgs struct {
	remAction models.ActionOpt
	ent       protoreflect.ProtoMessage
	pol       map[string]any
	params    map[string]any
	ruleName  string
}

func createTestRemArgs() *remediateArgs {
	return &remediateArgs{
		remAction: models.ActionOptOn,
		ent: &pb.Repository{
			Owner: repoOwner,
			Name:  repoName,
		},
		pol:    map[string]any{},
		params: map[string]any{},
	}
}

func TestIssueRemediate(t *testing.T) {
	t.Parallel()

	type newIssueRemediateArgs struct {
		issueRem   *pb.RuleType_Definition_Remediate_IssueRemediation
		actionType interfaces.ActionType
	}

	tests := []struct {
		name             string
		newRemArgs       *newIssueRemediateArgs
		remArgs          *remediateArgs
		mockSetup        func(*testing.T, *mockghclient.MockGitHub)
		cmd              interfaces.ActionCmd
		metadata         *json.RawMessage
		expectedErr      error
		wantInitErr      bool
		expectedMetadata json.RawMessage
	}{
		{
			name: "create an issue",
			newRemArgs: &newIssueRemediateArgs{
				issueRem: &pb.RuleType_Definition_Remediate_IssueRemediation{
					Title: issueTitle,
					Body:  issueBody,
					Labels: []string{
						"security",
						"automated",
					},
					Assignees: []string{
						"alice",
					},
				},
				actionType: TestActionTypeValid,
			},
			remArgs: createTestRemArgs(),
			mockSetup: func(_ *testing.T, mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					CreateIssue(
						gomock.Any(),
						repoOwner,
						repoName,
						issueTitle,
						issueBody,
						[]string{"security", "automated"},
						[]string{"alice"},
					).
					Return(
						&github.Issue{
							Number: github.Int(42),
						},
						nil,
					)
			},
			cmd:              interfaces.ActionCmdOn,
			metadata:         nil,
			expectedErr:      errors.ErrActionPending,
			expectedMetadata: json.RawMessage(`{"issue_number":42}`),
		},
		{
			name: "fail to create issue",
			newRemArgs: &newIssueRemediateArgs{
				issueRem:   defaultIssueRem(),
				actionType: TestActionTypeValid,
			},
			remArgs: createTestRemArgs(),
			mockSetup: func(_ *testing.T, mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					CreateIssue(
						gomock.Any(),
						repoOwner,
						repoName,
						issueTitle,
						issueBody,
						[]string{},
						[]string{},
					).
					Return(nil, fmt.Errorf("failed to create issue"))
			},
			cmd:              interfaces.ActionCmdOn,
			metadata:         nil,
			expectedErr:      errors.ErrActionFailed,
			expectedMetadata: nil,
		},
		{
			name: "issue already exists",
			newRemArgs: &newIssueRemediateArgs{
				issueRem:   defaultIssueRem(),
				actionType: TestActionTypeValid,
			},
			remArgs: createTestRemArgs(),
			mockSetup: func(_ *testing.T, _ *mockghclient.MockGitHub) {
				// Intentionally empty.
				// If CreateIssue() is called, gomock will fail the test.
			},
			cmd: interfaces.ActionCmdOn,
			metadata: func() *json.RawMessage {
				m := json.RawMessage(`{"issue_number":42}`)
				return &m
			}(),
			expectedErr:      errors.ErrActionPending,
			expectedMetadata: json.RawMessage(`{"issue_number":42}`),
		},
		{
			name: "close an issue",
			newRemArgs: &newIssueRemediateArgs{
				issueRem:   defaultIssueRem(),
				actionType: TestActionTypeValid,
			},
			remArgs: createTestRemArgs(),
			mockSetup: func(_ *testing.T, mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					CloseIssue(
						gomock.Any(),
						repoOwner,
						repoName,
						42,
						"",
					).
					Return(
						&github.Issue{
							Number: github.Int(42),
						},
						nil,
					)
			},
			cmd: interfaces.ActionCmdOff,
			metadata: func() *json.RawMessage {
				m := json.RawMessage(`{"issue_number":42}`)
				return &m
			}(),
			expectedErr:      errors.ErrActionSkipped,
			expectedMetadata: nil,
		},
		{
			name: "close already closed issue",
			newRemArgs: &newIssueRemediateArgs{
				issueRem:   defaultIssueRem(),
				actionType: TestActionTypeValid,
			},
			remArgs: createTestRemArgs(),
			mockSetup: func(_ *testing.T, mockGitHub *mockghclient.MockGitHub) {
				// GitHub returns 200 OK when closing an already closed issue.
				// This documents provider behavior; other code forges may differ.
				mockGitHub.EXPECT().
					CloseIssue(
						gomock.Any(),
						repoOwner,
						repoName,
						42,
						"",
					).
					Return(
						&github.Issue{
							Number: github.Int(42),
						},
						nil,
					)
			},
			cmd: interfaces.ActionCmdOff,
			metadata: func() *json.RawMessage {
				m := json.RawMessage(`{"issue_number":42}`)
				return &m
			}(),
			expectedErr:      errors.ErrActionSkipped,
			expectedMetadata: nil,
		},
		{
			name: "close issue without metadata",
			newRemArgs: &newIssueRemediateArgs{
				issueRem:   defaultIssueRem(),
				actionType: TestActionTypeValid,
			},
			remArgs: createTestRemArgs(),
			mockSetup: func(_ *testing.T, _ *mockghclient.MockGitHub) {
				// No expectations.
				// CloseIssue must NOT be called.
			},
			cmd:              interfaces.ActionCmdOff,
			metadata:         nil,
			expectedErr:      errors.ErrActionSkipped,
			expectedMetadata: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mockghclient.NewMockGitHub(ctrl)

			engine, err := NewIssueRemediate(
				tt.newRemArgs.actionType,
				tt.newRemArgs.issueRem,
				mockClient,
				tt.remArgs.remAction,
			)

			if tt.wantInitErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, engine)

			tt.mockSetup(t, mockClient)

			evalParams := &interfaces.EvalStatusParams{
				Rule: &models.RuleInstance{
					Def:    tt.remArgs.pol,
					Params: tt.remArgs.params,
					Name:   tt.remArgs.ruleName,
				},
			}

			evalParams.SetEvalResult(&interfaces2.EvaluationResult{})

			retMeta, err := engine.Do(
				context.Background(),
				tt.cmd,
				tt.remArgs.ent,
				evalParams,
				tt.metadata,
			)

			require.ErrorIs(t, err, tt.expectedErr)
			require.Equal(t, tt.expectedMetadata, retMeta)
		})
	}
}
