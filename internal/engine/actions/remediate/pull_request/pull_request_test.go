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

// Package pull_request provides the pull request remediation engine
package pull_request

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v53/github"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	mock_ghclient "github.com/stacklok/minder/internal/providers/github/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	ghApiUrl = "https://api.github.com"

	repoOwner = "stacklok"
	repoName  = "minder"

	refSha        = "f254eba2db416be8d94aa35bcf3a1c41b6a6926c"
	treeSha       = "f00cf9d55a642ec402f407dd7c3aaff69a17658b"
	branchFrom    = "dependabot/gomod"
	dependabotSha = "dependabot-sha"
	readmeSha     = "readme-sha"
	newTreeSha    = "new-tree-sha"
	newCommitSha  = "new-commit-sha"

	commitTitle = "Add Dependabot configuration for gomod"
	prBody      = `<!-- minder: pr-remediation-body: { "ContentSha": "1041e57c2fac284bdb7827ce55c6e3cb609e97b9" } -->

Adds Dependabot configuration for gomod`
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
		db.ProviderAccessToken{},
		"token",
	)
}

func dependabotPrRem() *pb.RuleType_Definition_Remediate_PullRequestRemediation {
	return &pb.RuleType_Definition_Remediate_PullRequestRemediation{
		Title: "Add Dependabot configuration for {{.Profile.package_ecosystem }}",
		Body:  "Adds Dependabot configuration for {{.Profile.package_ecosystem }}",
		Contents: []*pb.RuleType_Definition_Remediate_PullRequestRemediation_Content{
			{
				Path:    ".github/dependabot.yml",
				Content: "dependabot config for {{.Profile.package_ecosystem }}",
			},
			{
				Path:    "README.md",
				Content: "This project uses dependabot",
			},
		},
	}
}

type remediateArgs struct {
	remAction interfaces.ActionOpt
	ent       protoreflect.ProtoMessage
	pol       map[string]any
	params    map[string]any
}

func createTestRemArgs() *remediateArgs {
	return &remediateArgs{
		remAction: interfaces.ActionOptOn,
		ent: &pb.Repository{
			Owner: repoOwner,
			Name:  repoName,
		},
		pol: map[string]any{
			"package_ecosystem": "gomod",
			"schedule_interval": "30 4-6 * * *",
		},
		params: map[string]any{
			// explicitly test non-default branch
			"branch": "dependabot/gomod",
		},
	}
}

func happyPathMockSetup(mockGitHub *mock_ghclient.MockGitHub) {
	// no pull requst so far
	mockGitHub.EXPECT().
		ListPullRequests(gomock.Any(), repoOwner, repoName, gomock.Any()).Return([]*github.PullRequest{}, nil)
	mockGitHub.EXPECT().
		GetRef(gomock.Any(), repoOwner, repoName, refFromBranch(branchFrom)).
		Return(
			&github.Reference{
				Object: &github.GitObject{
					SHA: github.String(refSha),
				},
			},
			nil)
	mockGitHub.EXPECT().
		GetCommit(gomock.Any(), repoOwner, repoName, refSha).
		Return(
			&github.Commit{
				Tree: &github.Tree{
					SHA: github.String(treeSha),
				},
			},
			nil)
	mockGitHub.EXPECT().
		CreateBlob(gomock.Any(), repoOwner, repoName, &github.Blob{
			Content:  github.String("dependabot config for gomod"),
			Encoding: github.String("utf-8"),
		}).
		Return(
			&github.Blob{
				SHA: github.String(dependabotSha),
			},
			nil)
	mockGitHub.EXPECT().
		CreateBlob(gomock.Any(), repoOwner, repoName, &github.Blob{
			Content:  github.String("This project uses dependabot"),
			Encoding: github.String("utf-8"),
		}).
		Return(
			&github.Blob{
				SHA: github.String(readmeSha),
			},
			nil)
	mockGitHub.EXPECT().
		CreateTree(gomock.Any(), repoOwner, repoName, treeSha,
			[]*github.TreeEntry{
				{
					Path: github.String(".github/dependabot.yml"),
					Mode: github.String("100644"),
					Type: github.String("blob"),
					SHA:  github.String(dependabotSha),
				},
				{
					Path: github.String("README.md"),
					Mode: github.String("100644"),
					Type: github.String("blob"),
					SHA:  github.String(readmeSha),
				},
			}).
		Return(
			&github.Tree{
				SHA: github.String(newTreeSha),
			}, nil)
	mockGitHub.EXPECT().
		CreateCommit(gomock.Any(), repoOwner, repoName, commitTitle,
			&github.Tree{
				SHA: github.String(newTreeSha),
			}, refSha).
		Return(
			&github.Commit{
				SHA: github.String(newCommitSha),
			}, nil)

}

func TestPullRequestRemediate(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	type newPullRequestRemediateArgs struct {
		prRem      *pb.RuleType_Definition_Remediate_PullRequestRemediation
		pbuild     *providers.ProviderBuilder
		actionType interfaces.ActionType
	}

	tests := []struct {
		name        string
		newRemArgs  *newPullRequestRemediateArgs
		remArgs     *remediateArgs
		mockSetup   func(*mock_ghclient.MockGitHub)
		wantErr     bool
		wantInitErr bool
	}{
		{
			name: "open a PR",
			newRemArgs: &newPullRequestRemediateArgs{
				prRem:      dependabotPrRem(),
				pbuild:     testGithubProviderBuilder(ghApiUrl),
				actionType: TestActionTypeValid,
			},
			remArgs: createTestRemArgs(),
			mockSetup: func(mockGitHub *mock_ghclient.MockGitHub) {
				happyPathMockSetup(mockGitHub)

				mockGitHub.EXPECT().
					CreateRef(gomock.Any(), repoOwner, repoName,
						refFromBranch(branchBaseName(commitTitle)),
						newCommitSha).
					Return(nil, nil)
				mockGitHub.EXPECT().
					CreatePullRequest(
						gomock.Any(),
						repoOwner, repoName,
						commitTitle, prBody,
						refFromBranch(branchBaseName(commitTitle)), dflBranchTo).
					Return(nil, nil)
			},
		},
		{
			name: "update an existing PR branch with a force-push",
			newRemArgs: &newPullRequestRemediateArgs{
				prRem:      dependabotPrRem(),
				pbuild:     testGithubProviderBuilder(ghApiUrl),
				actionType: TestActionTypeValid,
			},
			remArgs: createTestRemArgs(),
			mockSetup: func(mockGitHub *mock_ghclient.MockGitHub) {
				happyPathMockSetup(mockGitHub)

				// tests that createRef detects that the branch already exists and updateRef force-pushes
				mockGitHub.EXPECT().
					CreateRef(gomock.Any(), repoOwner, repoName,
						refFromBranch(branchBaseName(commitTitle)),
						newCommitSha).
					Return(
						nil, &github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusUnprocessableEntity,
							},
						})
				mockGitHub.EXPECT().
					UpdateRef(gomock.Any(), repoOwner, repoName,
						refFromBranch(branchBaseName(commitTitle)),
						newCommitSha, true).
					Return(nil, nil)
				mockGitHub.EXPECT().
					CreatePullRequest(
						gomock.Any(),
						repoOwner, repoName,
						commitTitle, prBody,
						refFromBranch(branchBaseName(commitTitle)), dflBranchTo).
					Return(nil, nil)
			},
		},
		{
			name: "A PR with the same content already exists",
			newRemArgs: &newPullRequestRemediateArgs{
				prRem:      dependabotPrRem(),
				pbuild:     testGithubProviderBuilder(ghApiUrl),
				actionType: TestActionTypeValid,
			},
			remArgs: createTestRemArgs(),
			mockSetup: func(mockGitHub *mock_ghclient.MockGitHub) {
				mockGitHub.EXPECT().
					ListPullRequests(gomock.Any(), repoOwner, repoName, gomock.Any()).
					Return([]*github.PullRequest{
						{
							Body: github.String(prBody),
						},
					}, nil)
			},
		},
	}

	mockClient := mock_ghclient.NewMockGitHub(ctrl)
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			engine, err := NewPullRequestRemediate(tt.newRemArgs.actionType, tt.newRemArgs.prRem, tt.newRemArgs.pbuild)
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

			retMeta, err := engine.Do(context.Background(), interfaces.ActionCmdOn, tt.remArgs.remAction, tt.remArgs.ent, evalParams, nil, nil)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, retMeta, "expected nil metadata")
				return
			}

			require.NoError(t, err, "unexpected error running remediate engine")
		})
	}
}
