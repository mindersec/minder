// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package gh_branch_protect provides the github branch protection remediation engine
package gh_branch_protect

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/credentials"
	mock_ghclient "github.com/stacklok/minder/internal/providers/github/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	ghApiUrl         = "https://api.github.com"
	reviewCountPatch = `{"required_pull_request_reviews":{"required_approving_review_count":{{ .Profile.required_approving_review_count }}}}`

	repoOwner = "stacklok"
	repoName  = "minder"
)

var TestActionTypeValid interfaces.ActionType = "remediate-test"

func testGithubProviderBuilder(baseURL string) *providers.ProviderBuilder {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}

	definitionJSON := `{
		"github": {
			"endpoint": "` + baseURL + `"
		}
	}`

	return providers.NewProviderBuilder(
		&db.Provider{
			Name:       "github",
			Version:    provifv1.V1,
			Implements: []db.ProviderType{db.ProviderTypeGithub, db.ProviderTypeRest},
			Definition: json.RawMessage(definitionJSON),
		},
		sql.NullString{},
		false,
		credentials.NewGitHubTokenCredential("token"),
		&serverconfig.ProviderConfig{},
	)
}

type protectionRequestMatcher struct {
	exp *github.ProtectionRequest
}

func (m *protectionRequestMatcher) Matches(x interface{}) bool {
	req, ok := x.(*github.ProtectionRequest)
	if !ok {
		return false
	}

	if m.exp.AllowForcePushes != nil {
		if req.AllowForcePushes == nil {
			return false
		}

		if *req.AllowForcePushes != *m.exp.AllowForcePushes {
			return false
		}
	}

	if m.exp.RequiredStatusChecks != nil {
		if req.RequiredStatusChecks == nil {
			return false
		}

		if len(req.GetRequiredStatusChecks().GetContexts()) != len(m.exp.GetRequiredStatusChecks().GetContexts()) {
			return false
		}

		for i, c := range req.GetRequiredStatusChecks().GetContexts() {
			if c != m.exp.GetRequiredStatusChecks().GetContexts()[i] {
				return false
			}
		}

		if len(req.GetRequiredStatusChecks().GetChecks()) != len(m.exp.GetRequiredStatusChecks().GetChecks()) {
			return false
		}

		for i, c := range req.GetRequiredStatusChecks().GetChecks() {
			if c.Context != m.exp.GetRequiredStatusChecks().GetChecks()[i].Context ||
				*c.AppID != *m.exp.GetRequiredStatusChecks().GetChecks()[i].AppID {
				return false
			}
		}
	}

	return req.RequiredPullRequestReviews.RequiredApprovingReviewCount == m.exp.RequiredPullRequestReviews.RequiredApprovingReviewCount &&
		req.AllowDeletions == m.exp.AllowDeletions
}

func (m *protectionRequestMatcher) String() string {
	return fmt.Sprintf("is equivalent to proto %+v", m.exp)
}

func eqProtectionRequest(exp *github.ProtectionRequest) gomock.Matcher {
	return &protectionRequestMatcher{exp}
}

func TestBranchProtectionRemediate(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	type remediateArgs struct {
		remAction interfaces.ActionOpt
		ent       protoreflect.ProtoMessage
		pol       map[string]any
		params    map[string]any
	}

	type newBranchProtectionRemediateArgs struct {
		ghp        *pb.RuleType_Definition_Remediate_GhBranchProtectionType
		pbuild     *providers.ProviderBuilder
		actionType interfaces.ActionType
	}

	tests := []struct {
		name        string
		newRemArgs  *newBranchProtectionRemediateArgs
		remArgs     *remediateArgs
		mockSetup   func(*mock_ghclient.MockGitHub)
		wantErr     bool
		wantInitErr bool
	}{
		{
			name: "invalid action type",
			newRemArgs: &newBranchProtectionRemediateArgs{
				ghp: &pb.RuleType_Definition_Remediate_GhBranchProtectionType{
					Patch: reviewCountPatch,
				},
				pbuild:     testGithubProviderBuilder(ghApiUrl),
				actionType: "",
			},
			mockSetup: func(_ *mock_ghclient.MockGitHub) {
			},
			remArgs: &remediateArgs{
				remAction: interfaces.ActionOptOn,
				ent: &pb.Repository{
					Owner: repoOwner,
					Name:  repoName,
				},
				pol: map[string]any{
					"required_approving_review_count": 2,
				},
				params: map[string]any{
					"branch": "main",
				},
			},
			wantInitErr: true,
		},
		{
			name: "No protection was in place",
			newRemArgs: &newBranchProtectionRemediateArgs{
				ghp: &pb.RuleType_Definition_Remediate_GhBranchProtectionType{
					Patch: reviewCountPatch,
				},
				pbuild:     testGithubProviderBuilder(ghApiUrl),
				actionType: TestActionTypeValid,
			},
			remArgs: &remediateArgs{
				remAction: interfaces.ActionOptOn,
				ent: &pb.Repository{
					Owner: repoOwner,
					Name:  repoName,
				},
				pol: map[string]any{
					"required_approving_review_count": 2,
				},
				params: map[string]any{
					"branch": "main",
				},
			},
			mockSetup: func(mockGitHub *mock_ghclient.MockGitHub) {
				mockGitHub.EXPECT().
					GetBranchProtection(gomock.Any(), repoOwner, repoName, "main").
					Return(nil, github.ErrBranchNotProtected)
				mockGitHub.EXPECT().
					UpdateBranchProtection(gomock.Any(), repoOwner, repoName, "main",
						// nested pointers to structs confuse gmock
						eqProtectionRequest(
							&github.ProtectionRequest{
								RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
									RequiredApprovingReviewCount: 2,
								},
							},
						),
					).
					Return(nil)
			},
		},
		{
			name: "Some protection was in place, remediator merges the patch",
			newRemArgs: &newBranchProtectionRemediateArgs{
				ghp: &pb.RuleType_Definition_Remediate_GhBranchProtectionType{
					Patch: reviewCountPatch,
				},
				pbuild:     testGithubProviderBuilder(ghApiUrl),
				actionType: TestActionTypeValid,
			},
			remArgs: &remediateArgs{
				remAction: interfaces.ActionOptOn,
				ent: &pb.Repository{
					Owner: repoOwner,
					Name:  repoName,
				},
				pol: map[string]any{
					"required_approving_review_count": 2,
				},
				params: map[string]any{
					"branch": "main",
				},
			},
			mockSetup: func(mockGitHub *mock_ghclient.MockGitHub) {
				mockGitHub.EXPECT().
					GetBranchProtection(gomock.Any(), repoOwner, repoName, "main").
					Return(
						&github.Protection{
							RequiredPullRequestReviews: &github.PullRequestReviewsEnforcement{
								RequiredApprovingReviewCount: 1,
							},
							AllowForcePushes: &github.AllowForcePushes{Enabled: true},
							RequiredStatusChecks: &github.RequiredStatusChecks{
								Contexts: &[]string{"ci"},
								Checks: &[]*github.RequiredStatusCheck{
									{
										Context: "ci",
										AppID:   github.Int64(1234),
									},
								},
							},
						},
						nil)

				mockGitHub.EXPECT().
					UpdateBranchProtection(gomock.Any(), repoOwner, repoName, "main",
						// nested pointers to structs confuse gmock
						eqProtectionRequest(
							&github.ProtectionRequest{
								RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
									RequiredApprovingReviewCount: 2,
								},
								AllowForcePushes: github.Bool(true),
								RequiredStatusChecks: &github.RequiredStatusChecks{
									Checks: &[]*github.RequiredStatusCheck{
										{
											Context: "ci",
											AppID:   github.Int64(1234),
										},
									},
								},
							},
						),
					).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := mock_ghclient.NewMockGitHub(ctrl)

			engine, err := NewGhBranchProtectRemediator(tt.newRemArgs.actionType, tt.newRemArgs.ghp, tt.newRemArgs.pbuild)
			if tt.wantInitErr {
				require.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "unexpected error creating remediate engine")
			// TODO(jakub): providerBuilder should be an interface so we can pass in mock more easily
			engine.cli = mockClient

			require.NoError(t, err, "unexpected error creating remediate engine")
			require.NotNil(t, engine, "expected non-nil remediate engine")

			tt.mockSetup(mockClient)

			structPol, err := structpb.NewStruct(tt.remArgs.pol)
			if err != nil {
				fmt.Printf("Error creating Struct: %v\n", err)
				return
			}
			structParams, err := structpb.NewStruct(tt.remArgs.params)
			if err != nil {
				fmt.Printf("Error creating Struct: %v\n", err)
				return
			}
			evalParams := &interfaces.EvalStatusParams{
				Rule: &pb.Profile_Rule{
					Def:    structPol,
					Params: structParams,
				},
			}

			retMeta, err := engine.Do(context.Background(), interfaces.ActionCmdOn, tt.remArgs.remAction, tt.remArgs.ent, evalParams, nil)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, retMeta, "expected nil metadata")
				return
			}

			require.NoError(t, err, "unexpected error running remediate engine")
		})
	}
}
