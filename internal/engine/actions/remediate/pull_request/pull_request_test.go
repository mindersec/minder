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
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/google/go-github/v60/github"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/credentials"
	mock_ghclient "github.com/stacklok/minder/internal/providers/github/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	ghApiUrl = "https://api.github.com"

	repoOwner = "stacklok"
	repoName  = "minder"

	commitTitle = "Add Dependabot configuration for gomod"
	prBody      = `Adds Dependabot configuration for gomod`

	authorLogin = "stacklok-bot"
	authorEmail = "bot@stacklok.com"

	frizbeeCommitTitle        = "Replace tags with sha"
	frizbeePrBody             = `This PR replaces tags with sha`
	frizbeePrBodyWithExcludes = `This PR replaces tags with sha`

	actionWithTags = `
on:
  workflow_call:
jobs:
  build:
    name: Verify build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4 # v3.5.0
      - name: Extract version of Go to use
        run: echo "GOVERSION=$(sed -n 's/^go \([0-9.]*\)/\1/p' go.mod)" >> $GITHUB_ENV
      - uses: actions/setup-go@v5 # v4.0.0
        with:
          go-version-file: 'go.mod'
      - name: build
        run: make build
`
	checkoutV4Ref = "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	setupV5Ref    = "1041e57c2fac284bdb7827ce55c6e3cb609e97b9"
)

var TestActionTypeValid interfaces.ActionType = "remediate-test"

func testGithubProviderBuilder() *providers.ProviderBuilder {
	baseURL := ghApiUrl + "/"

	definitionJSON := `{
		"github": {
			"endpoint": "` + baseURL + `"
		}
	}`

	return providers.NewProviderBuilder(
		&db.Provider{
			Name:       "github",
			Version:    provifv1.V1,
			Implements: []db.ProviderType{db.ProviderTypeGithub, db.ProviderTypeRest, db.ProviderTypeGit},
			Definition: json.RawMessage(definitionJSON),
		},
		sql.NullString{},
		credentials.NewGitHubTokenCredential("token"),
		&serverconfig.ProviderConfig{},
	)
}

func dependabotPrRem() *pb.RuleType_Definition_Remediate_PullRequestRemediation {
	return &pb.RuleType_Definition_Remediate_PullRequestRemediation{
		Method: "minder.content",
		Title:  "Add Dependabot configuration for {{.Profile.package_ecosystem }}",
		Body:   "Adds Dependabot configuration for {{.Profile.package_ecosystem }}",
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

func frizbeePrRem() *pb.RuleType_Definition_Remediate_PullRequestRemediation {
	return &pb.RuleType_Definition_Remediate_PullRequestRemediation{
		Method: "minder.actions.replace_tags_with_sha",
		Title:  frizbeeCommitTitle,
		Body:   "This PR replaces tags with sha",
	}
}

func frizbeePrRemWithExcludes(e []string) *pb.RuleType_Definition_Remediate_PullRequestRemediation {
	return &pb.RuleType_Definition_Remediate_PullRequestRemediation{
		Method: "minder.actions.replace_tags_with_sha",
		Title:  frizbeeCommitTitle,
		Body:   "This PR replaces tags with sha",
		ActionsReplaceTagsWithSha: &pb.RuleType_Definition_Remediate_PullRequestRemediation_ActionsReplaceTagsWithSha{
			Exclude: e,
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

func createTestRemArgsWithExcludes() *remediateArgs {
	return &remediateArgs{
		remAction: interfaces.ActionOptOn,
		ent: &pb.Repository{
			Owner: repoOwner,
			Name:  repoName,
		},
		pol: map[string]any{
			"exclude": []any{"actions/setup-go@v5"},
		},
		params: map[string]any{
			// explicitly test non-default branch
			"branch": "dependabot/gomod",
		},
	}
}

func happyPathMockSetup(mockGitHub *mock_ghclient.MockGitHub) {
	// no pull request so far
	//mockGitHub.EXPECT().
	//	ListPullRequests(gomock.Any(), repoOwner, repoName, gomock.Any()).Return([]*github.PullRequest{}, nil)
	mockGitHub.EXPECT().
		GetName(gomock.Any()).Return("stacklok-bot", nil)
	mockGitHub.EXPECT().
		GetPrimaryEmail(gomock.Any()).Return("test@stacklok.com", nil)
	mockGitHub.EXPECT().
		AddAuthToPushOptions(gomock.Any(), gomock.Any()).Return(nil)
}

func resolveActionMockSetup(t *testing.T, mockGitHub *mock_ghclient.MockGitHub, url, ref string) {
	t.Helper()

	mockGitHub.EXPECT().
		NewRequest(http.MethodGet, url, nil).
		Return(&http.Request{}, nil)

	checkoutRef := github.Reference{
		Object: &github.GitObject{
			SHA: github.String(ref),
		},
	}
	jsonCheckoutRef, err := json.Marshal(checkoutRef)
	require.NoError(t, err, "unexpected error marshalling checkout ref")

	mockGitHub.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(&http.Response{
			Body:       io.NopCloser(bytes.NewBuffer(jsonCheckoutRef)),
			StatusCode: http.StatusOK,
		}, nil)

}
func mockRepoSetup(t *testing.T, postSetupHooks ...hookFunc) (*git.Repository, error) {
	t.Helper()

	upstream, err := mockUpstreamSetup(t, postSetupHooks...)
	if err != nil {
		return nil, err
	}

	clone, err := mockCloneSetup(upstream)
	if err != nil {
		return nil, err
	}

	return clone, nil
}

type hookFunc func(*git.Repository) error

func mockUpstreamSetup(t *testing.T, postSetupHooks ...hookFunc) (*git.Repository, error) {
	t.Helper()

	tmpdir := t.TempDir()
	fsStorer := filesystem.NewStorage(osfs.New(tmpdir), nil)
	// we create an OS filesystem with a bound OS so that the git repo can be
	// references as an upstream remote using file:// URL
	fs := osfs.New(tmpdir, osfs.WithBoundOS())

	// initialize the repo with a commit or else creating a branch will fail
	r, err := git.Init(fsStorer, fs)
	if err != nil {
		return nil, err
	}

	wt, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	// create a dummy commit or else checking out a branch fails
	// TODO(jakub): is this something to handle in the engine?
	f, err := wt.Filesystem.Create(".gitignore")
	if err != nil {
		return nil, err
	}

	_, err = f.Write([]byte("This is a test repo"))
	if err != nil {
		return nil, err
	}

	_, err = wt.Add(".gitignore")
	if err != nil {
		return nil, err
	}

	f, err = wt.Filesystem.Create(".github/workflows/build.yml")
	if err != nil {
		return nil, err
	}

	_, err = f.Write([]byte(actionWithTags))
	if err != nil {
		return nil, err
	}

	_, err = wt.Add(".github/workflows/build.yml")
	if err != nil {
		return nil, err
	}

	_, err = wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorLogin,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	for _, hook := range postSetupHooks {
		if err := hook(r); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func mockCloneSetup(upstream *git.Repository) (*git.Repository, error) {
	mfs := memfs.New()
	memStorer := memory.NewStorage()

	upstreamWt, err := upstream.Worktree()
	if err != nil {
		return nil, err
	}

	upstreamRoot := upstreamWt.Filesystem.Root()
	clone, err := git.Clone(memStorer, mfs, &git.CloneOptions{
		URL: "file://" + upstreamRoot,
	})
	if err != nil {
		return nil, err
	}

	return clone, nil
}

func defaultMockRepoSetup(t *testing.T) (*git.Repository, error) {
	t.Helper()

	return mockRepoSetup(t)
}

func mockRepoSetupWithBranch(t *testing.T) (*git.Repository, error) {
	t.Helper()

	return mockRepoSetup(t, func(repo *git.Repository) error {
		headRef, err := repo.Head()
		if err != nil {
			return err
		}

		// for some reason just checkout the new branch was throwing a panic
		// this is just a convoluted way of creating a new branch
		ref := plumbing.NewHashReference(
			plumbing.ReferenceName(
				refFromBranch(branchBaseName(commitTitle)),
			),
			headRef.Hash())

		return repo.Storer.SetReference(ref)
	})
}

func TestPullRequestRemediate(t *testing.T) {
	t.Parallel()

	type newPullRequestRemediateArgs struct {
		prRem      *pb.RuleType_Definition_Remediate_PullRequestRemediation
		pbuild     *providers.ProviderBuilder
		actionType interfaces.ActionType
	}

	tests := []struct {
		name             string
		newRemArgs       *newPullRequestRemediateArgs
		remArgs          *remediateArgs
		repoSetup        func(*testing.T) (*git.Repository, error)
		mockSetup        func(*testing.T, *mock_ghclient.MockGitHub)
		expectedErr      error
		wantInitErr      bool
		expectedMetadata json.RawMessage
	}{
		{
			name: "open a PR",
			newRemArgs: &newPullRequestRemediateArgs{
				prRem:      dependabotPrRem(),
				pbuild:     testGithubProviderBuilder(),
				actionType: TestActionTypeValid,
			},
			remArgs:   createTestRemArgs(),
			repoSetup: defaultMockRepoSetup,
			mockSetup: func(_ *testing.T, mockGitHub *mock_ghclient.MockGitHub) {
				happyPathMockSetup(mockGitHub)

				mockGitHub.EXPECT().
					CreatePullRequest(
						gomock.Any(),
						repoOwner, repoName,
						commitTitle, prBody,
						refFromBranch(branchBaseName(commitTitle)), dflBranchTo).
					Return(&github.PullRequest{Number: github.Int(42)}, nil)
			},
			expectedErr:      errors.ErrActionPending,
			expectedMetadata: json.RawMessage(`{"pr_number":42}`),
		},
		{
			name: "fail to open a PR",
			newRemArgs: &newPullRequestRemediateArgs{
				prRem:      dependabotPrRem(),
				pbuild:     testGithubProviderBuilder(),
				actionType: TestActionTypeValid,
			},
			remArgs:   createTestRemArgs(),
			repoSetup: defaultMockRepoSetup,
			mockSetup: func(_ *testing.T, mockGitHub *mock_ghclient.MockGitHub) {
				happyPathMockSetup(mockGitHub)

				mockGitHub.EXPECT().
					CreatePullRequest(
						gomock.Any(),
						repoOwner, repoName,
						commitTitle, prBody,
						refFromBranch(branchBaseName(commitTitle)), dflBranchTo).
					Return(nil, fmt.Errorf("failed to create PR"))
			},
			expectedErr:      errors.ErrActionFailed,
			expectedMetadata: json.RawMessage(nil),
		},
		{
			name: "update an existing PR branch with a force-push",
			newRemArgs: &newPullRequestRemediateArgs{
				prRem:      dependabotPrRem(),
				pbuild:     testGithubProviderBuilder(),
				actionType: TestActionTypeValid,
			},
			remArgs:   createTestRemArgs(),
			repoSetup: mockRepoSetupWithBranch,
			mockSetup: func(_ *testing.T, mockGitHub *mock_ghclient.MockGitHub) {
				happyPathMockSetup(mockGitHub)

				mockGitHub.EXPECT().
					CreatePullRequest(
						gomock.Any(),
						repoOwner, repoName,
						commitTitle, prBody,
						refFromBranch(branchBaseName(commitTitle)), dflBranchTo).
					Return(&github.PullRequest{Number: github.Int(41)}, nil)
			},
			expectedErr:      errors.ErrActionPending,
			expectedMetadata: json.RawMessage(`{"pr_number":41}`),
		},
		//{
		//	name: "A PR with the same content already exists",
		//	newRemArgs: &newPullRequestRemediateArgs{
		//		prRem:      dependabotPrRem(),
		//		pbuild:     testGithubProviderBuilder(),
		//		actionType: TestActionTypeValid,
		//	},
		//	remArgs:   createTestRemArgs(),
		//	repoSetup: defaultMockRepoSetup,
		//	mockSetup: func(_ *testing.T, mockGitHub *mock_ghclient.MockGitHub) {
		//		mockGitHub.EXPECT().
		//			ListPullRequests(gomock.Any(), repoOwner, repoName, gomock.Any()).
		//			Return([]*github.PullRequest{
		//				{
		//					Body: github.String(prBody),
		//				},
		//			}, nil)
		//	},
		//},
		//
		//{
		//	name: "A branch for this PR already exists, shouldn't open a new PR, but only update the branch",
		//	newRemArgs: &newPullRequestRemediateArgs{
		//		prRem:      dependabotPrRem(),
		//		pbuild:     testGithubProviderBuilder(),
		//		actionType: TestActionTypeValid,
		//	},
		//	remArgs:   createTestRemArgs(),
		//	repoSetup: defaultMockRepoSetup,
		//	mockSetup: func(_ *testing.T, mockGitHub *mock_ghclient.MockGitHub) {
		//		// no pull requst so far
		//		mockGitHub.EXPECT().
		//			ListPullRequests(gomock.Any(), repoOwner, repoName, gomock.Any()).Return([]*github.PullRequest{}, nil)
		//		// we need to get the user information and update the branch
		//		mockGitHub.EXPECT().
		//			GetName(gomock.Any()).Return("stacklok-bot", nil)
		//		// likewise we need to update the branch with a valid e-mail
		//		mockGitHub.EXPECT().
		//			GetPrimaryEmail(gomock.Any()).Return("test@stacklok.com", nil)
		//		mockGitHub.EXPECT().
		//			AddAuthToPushOptions(gomock.Any(), gomock.Any()).Return(nil)
		//		// this is the last call we expect to make. It returns existing PRs from this branch, so we
		//		// stop after having updated the branch
		//		mockGitHub.EXPECT().
		//			ListPullRequests(gomock.Any(), repoOwner, repoName, gomock.Any()).Return([]*github.PullRequest{
		//			// it doesn't matter what we return here, we just need to return a non-empty list
		//			{
		//				Number: github.Int(1),
		//			},
		//		}, nil)
		//	},
		//},
		{
			name: "resolve tags using frizbee",
			newRemArgs: &newPullRequestRemediateArgs{
				prRem:      frizbeePrRem(),
				pbuild:     testGithubProviderBuilder(),
				actionType: TestActionTypeValid,
			},
			remArgs:   createTestRemArgs(),
			repoSetup: defaultMockRepoSetup,
			mockSetup: func(t *testing.T, mockGitHub *mock_ghclient.MockGitHub) {
				t.Helper()

				happyPathMockSetup(mockGitHub)

				resolveActionMockSetup(t, mockGitHub, "repos/actions/checkout/git/refs/tags/v4", checkoutV4Ref)
				resolveActionMockSetup(t, mockGitHub, "repos/actions/setup-go/git/refs/tags/v5", setupV5Ref)

				mockGitHub.EXPECT().
					CreatePullRequest(
						gomock.Any(),
						repoOwner, repoName,
						frizbeeCommitTitle, frizbeePrBody,
						refFromBranch(branchBaseName(frizbeeCommitTitle)), dflBranchTo).
					Return(&github.PullRequest{Number: github.Int(40)}, nil)
			},
			expectedErr:      errors.ErrActionPending,
			expectedMetadata: json.RawMessage(`{"pr_number":40}`),
		},
		{
			name: "resolve tags using frizbee with excludes",
			newRemArgs: &newPullRequestRemediateArgs{
				prRem:      frizbeePrRemWithExcludes([]string{"actions/setup-go@v5"}),
				pbuild:     testGithubProviderBuilder(),
				actionType: TestActionTypeValid,
			},
			remArgs:   createTestRemArgs(),
			repoSetup: defaultMockRepoSetup,
			mockSetup: func(t *testing.T, mockGitHub *mock_ghclient.MockGitHub) {
				t.Helper()

				happyPathMockSetup(mockGitHub)

				resolveActionMockSetup(t, mockGitHub, "repos/actions/checkout/git/refs/tags/v4", checkoutV4Ref)
				mockGitHub.EXPECT().
					CreatePullRequest(
						gomock.Any(),
						repoOwner, repoName,
						frizbeeCommitTitle, frizbeePrBodyWithExcludes,
						refFromBranch(branchBaseName(frizbeeCommitTitle)), dflBranchTo).
					Return(&github.PullRequest{Number: github.Int(43)}, nil)
			},
			expectedErr:      errors.ErrActionPending,
			expectedMetadata: json.RawMessage(`{"pr_number":43}`),
		},
		{
			name: "resolve tags using frizbee with excludes from rule",
			newRemArgs: &newPullRequestRemediateArgs{
				prRem:      frizbeePrRem(),
				pbuild:     testGithubProviderBuilder(),
				actionType: TestActionTypeValid,
			},
			remArgs:   createTestRemArgsWithExcludes(),
			repoSetup: defaultMockRepoSetup,
			mockSetup: func(t *testing.T, mockGitHub *mock_ghclient.MockGitHub) {
				t.Helper()

				happyPathMockSetup(mockGitHub)

				resolveActionMockSetup(t, mockGitHub, "repos/actions/checkout/git/refs/tags/v4", checkoutV4Ref)

				mockGitHub.EXPECT().
					CreatePullRequest(
						gomock.Any(),
						repoOwner, repoName,
						frizbeeCommitTitle, frizbeePrBodyWithExcludes,
						refFromBranch(branchBaseName(frizbeeCommitTitle)), dflBranchTo).
					Return(&github.PullRequest{Number: github.Int(44)}, nil)
			},
			expectedErr:      errors.ErrActionPending,
			expectedMetadata: json.RawMessage(`{"pr_number":44}`),
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

			mockClient := mock_ghclient.NewMockGitHub(ctrl)

			engine, err := NewPullRequestRemediate(tt.newRemArgs.actionType, tt.newRemArgs.prRem, tt.newRemArgs.pbuild)
			if tt.wantInitErr {
				require.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "unexpected error creating remediate engine")
			// TODO(jakub): providerBuilder should be an interface so we can pass in mock more easily
			engine.ghCli = mockClient

			require.NoError(t, err, "unexpected error creating remediate engine")
			require.NotNil(t, engine, "expected non-nil remediate engine")

			tt.mockSetup(t, mockClient)

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

			testrepo, err := tt.repoSetup(t)
			require.NoError(t, err, "unexpected error creating test repo")
			testWt, err := testrepo.Worktree()
			require.NoError(t, err, "unexpected error creating test worktree")

			evalParams.SetIngestResult(
				&interfaces.Result{
					Fs:     testWt.Filesystem,
					Storer: testrepo.Storer,
				})
			retMeta, err := engine.Do(context.Background(),
				interfaces.ActionCmdOn,
				tt.remArgs.remAction,
				tt.remArgs.ent,
				evalParams,
				nil)

			require.ErrorIs(t, err, tt.expectedErr, "expected error")
			require.Equal(t, tt.expectedMetadata, retMeta)
		})
	}
}
