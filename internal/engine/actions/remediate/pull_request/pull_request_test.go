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
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/golang/mock/gomock"
	"github.com/google/go-github/v56/github"
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

	commitTitle = "Add Dependabot configuration for gomod"
	prBody      = `<!-- minder: pr-remediation-body: { "ContentSha": "1041e57c2fac284bdb7827ce55c6e3cb609e97b9" } -->

Adds Dependabot configuration for gomod`

	authorLogin = "stacklok-bot"
	authorEmail = "bot@stacklok.com"
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
			Implements: []db.ProviderType{db.ProviderTypeGithub, db.ProviderTypeRest, db.ProviderTypeGit},
			Definition: json.RawMessage(definitionJSON),
		},
		db.ProviderAccessToken{},
		"token",
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
	mockGitHub.EXPECT().
		GetClient().Return(&github.Client{})
	// no pull requst so far
	mockGitHub.EXPECT().
		ListPullRequests(gomock.Any(), repoOwner, repoName, gomock.Any()).Return([]*github.PullRequest{}, nil)
	mockGitHub.EXPECT().
		GetAuthenticatedUser(gomock.Any()).Return(&github.User{
		Email: github.String("test@stacklok.com"),
		Login: github.String("stacklok-bot"),
	}, nil)
	mockGitHub.EXPECT().
		GetToken().Return("token")
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
		repoSetup   func(*testing.T) (*git.Repository, error)
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
			remArgs:   createTestRemArgs(),
			repoSetup: defaultMockRepoSetup,
			mockSetup: func(mockGitHub *mock_ghclient.MockGitHub) {
				happyPathMockSetup(mockGitHub)

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
			remArgs:   createTestRemArgs(),
			repoSetup: mockRepoSetupWithBranch,
			mockSetup: func(mockGitHub *mock_ghclient.MockGitHub) {
				happyPathMockSetup(mockGitHub)

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
			remArgs:   createTestRemArgs(),
			repoSetup: defaultMockRepoSetup,
			mockSetup: func(mockGitHub *mock_ghclient.MockGitHub) {
				mockGitHub.EXPECT().
					GetClient().Return(&github.Client{})
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
			engine.ghCli = mockClient

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
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, retMeta, "expected nil metadata")
				return
			}

			require.NoError(t, err, "unexpected error running remediate engine")
		})
	}
}
