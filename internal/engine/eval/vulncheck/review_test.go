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

// Package vulncheck provides the vulnerability check evaluator
package vulncheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-github/v56/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mock_ghclient "github.com/stacklok/minder/internal/providers/github/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	githubSubmitterID = 144222806
	githubMinderID    = 123456789

	minderReviewID = 987654321

	commitSHA = "27d6810b861c81e8c61e09c651875f5a976781d1"
)

func TestReviewPrHandlerNoVulnerabilities(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_ghclient.NewMockGitHub(ctrl)
	pr := &pb.PullRequest{
		Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/43",
		CommitSha: commitSHA,
		Number:    43,
		RepoOwner: "jakubtestorg",
		RepoName:  "bad-npm",
		AuthorId:  githubSubmitterID,
	}

	mockClient.EXPECT().GetAuthenticatedUser(gomock.Any()).Return(&github.User{
		ID: github.Int64(githubSubmitterID),
	}, nil)
	handler, err := newReviewPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	expBody, err := createReviewBody(noVulsFoundText)
	require.NoError(t, err)
	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("COMMENT"),
			Body:     github.String(expBody),
			Comments: make([]*github.DraftReviewComment, 0),
		})
	err = handler.submit(context.Background())
	require.NoError(t, err)
}

func TestReviewPrHandlerVulnerabilitiesDifferentIdentities(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_ghclient.NewMockGitHub(ctrl)
	pr := &pb.PullRequest{
		Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/43",
		CommitSha: commitSHA,
		Number:    43,
		RepoOwner: "jakubtestorg",
		RepoName:  "bad-npm",
		AuthorId:  githubSubmitterID,
	}

	mockClient.EXPECT().GetAuthenticatedUser(gomock.Any()).Return(&github.User{
		ID: github.Int64(githubMinderID),
	}, nil)
	handler, err := newReviewPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`+    "mongodb": {
+      "version": "5.1.0",
+      }`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	dep := &pb.PrDependencies_ContextualDependency{
		Dep: &pb.Dependency{
			Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_NPM,
			Name:      "mongodb",
			Version:   "0.5.0",
		},
		File: &pb.PrDependencies_ContextualDependency_FilePatch{
			Name:     "package-lock.json",
			PatchUrl: server.URL,
		},
	}

	patchPackage := &packageJson{
		Name:    "mongodb",
		Version: "0.6.0",
		Dist: struct {
			Integrity string `json:"integrity"`
			Tarball   string `json:"tarball"`
		}{
			Integrity: "sha512-+1+2+3+4+5+6+7+8+9+0",
			Tarball:   "https://registry.npmjs.org/mongodb/-/mongodb-0.6.0.tgz",
		},
	}
	mockClient.EXPECT().
		NewRequest("GET", server.URL, nil).
		Return(http.NewRequest("GET", server.URL, nil))
	mockClient.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`"%s": {`, patchPackage.Name))),
		}, nil)

	err = handler.trackVulnerableDep(context.TODO(), dep, nil, patchPackage)
	require.NoError(t, err)

	expBody, err := createReviewBody(vulnsFoundText)
	require.NoError(t, err)
	expCommentBody := reviewBodyWithSuggestion(patchPackage.IndentedString(0, fmt.Sprintf(`"%s": {`, patchPackage.Name), nil))

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("REQUEST_CHANGES"),
			Body:     github.String(expBody),
			Comments: []*github.DraftReviewComment{
				{
					Path:      github.String(dep.File.Name),
					StartLine: github.Int(1),
					Line:      github.Int(4),
					Body:      github.String(expCommentBody),
				},
			},
		})
	err = handler.submit(context.Background())
	require.NoError(t, err)
}

func TestReviewPrHandlerVulnerabilitiesWithNoPatchVersion(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_ghclient.NewMockGitHub(ctrl)
	pr := &pb.PullRequest{
		Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/43",
		CommitSha: commitSHA,
		Number:    43,
		RepoOwner: "jakubtestorg",
		RepoName:  "bad-npm",
		AuthorId:  githubSubmitterID,
	}

	mockClient.EXPECT().GetAuthenticatedUser(gomock.Any()).Return(&github.User{
		ID: github.Int64(githubMinderID),
	}, nil)
	handler, err := newReviewPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`+    "mongodb": {
+      "version": "5.1.0",
+      }`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	dep := &pb.PrDependencies_ContextualDependency{
		Dep: &pb.Dependency{
			Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_NPM,
			Name:      "mongodb",
			Version:   "0.5.0",
		},
		File: &pb.PrDependencies_ContextualDependency_FilePatch{
			Name:     "package-lock.json",
			PatchUrl: server.URL,
		},
	}

	patchPackage := &packageJson{}
	mockClient.EXPECT().
		NewRequest("GET", server.URL, nil).
		Return(http.NewRequest("GET", server.URL, nil))
	mockClient.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`"%s": {`, patchPackage.Name))),
		}, nil)

	err = handler.trackVulnerableDep(context.TODO(), dep, nil, patchPackage)
	require.NoError(t, err)

	expBody, err := createReviewBody(vulnsFoundText)
	require.NoError(t, err)
	expCommentBody := vulnFoundWithNoPatch

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("REQUEST_CHANGES"),
			Body:     github.String(expBody),
			Comments: []*github.DraftReviewComment{
				{
					Path: github.String(dep.File.Name),
					Line: github.Int(1),
					Body: github.String(expCommentBody),
				},
			},
		})
	err = handler.submit(context.Background())
	require.NoError(t, err)
}

func TestReviewPrHandlerVulnerabilitiesDismissReview(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_ghclient.NewMockGitHub(ctrl)
	pr := &pb.PullRequest{
		Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/43",
		CommitSha: commitSHA,
		Number:    43,
		RepoOwner: "jakubtestorg",
		RepoName:  "bad-npm",
		AuthorId:  githubSubmitterID,
	}

	mockClient.EXPECT().GetAuthenticatedUser(gomock.Any()).Return(&github.User{
		ID: github.Int64(githubSubmitterID),
	}, nil)
	handler, err := newReviewPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	expBody, err := createReviewBody(noVulsFoundText)
	require.NoError(t, err)

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{
			{
				ID:   github.Int64(minderReviewID),
				Body: github.String(reviewBodyMagicComment),
			},
		}, nil)

	mockClient.EXPECT().DismissReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), int64(minderReviewID),
		&github.PullRequestReviewDismissalRequest{
			Message: github.String(reviewBodyDismissCommentText),
		})

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("COMMENT"),
			Body:     github.String(expBody),
			Comments: make([]*github.DraftReviewComment, 0),
		})
	err = handler.submit(context.Background())
	require.NoError(t, err)
}

func TestCommitStatusHandlerNoVulnerabilities(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_ghclient.NewMockGitHub(ctrl)
	pr := &pb.PullRequest{
		Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/43",
		CommitSha: commitSHA,
		Number:    43,
		RepoOwner: "jakubtestorg",
		RepoName:  "bad-npm",
		AuthorId:  githubSubmitterID,
	}

	mockClient.EXPECT().GetAuthenticatedUser(gomock.Any()).Return(&github.User{
		ID: github.Int64(githubSubmitterID),
	}, nil)
	handler, err := newCommitStatusPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	expBody, err := createReviewBody(noVulsFoundText)
	require.NoError(t, err)
	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("COMMENT"),
			Body:     github.String(expBody),
			Comments: make([]*github.DraftReviewComment, 0),
		})

	mockClient.EXPECT().SetCommitStatus(gomock.Any(), pr.RepoOwner, pr.RepoName, commitSHA, &github.RepoStatus{
		State:       github.String("success"),
		Description: github.String(noVulsFoundText),
		Context:     github.String(commitStatusContext),
	})

	err = handler.submit(context.Background())
	require.NoError(t, err)
}

func TestCommitStatusPrHandlerWithVulnerabilities(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_ghclient.NewMockGitHub(ctrl)
	pr := &pb.PullRequest{
		Url:       "https://api.github.com/repos/jakubtestorg/bad-npm/pulls/43",
		CommitSha: commitSHA,
		Number:    43,
		RepoOwner: "jakubtestorg",
		RepoName:  "bad-npm",
		AuthorId:  githubSubmitterID,
	}

	mockClient.EXPECT().GetAuthenticatedUser(gomock.Any()).Return(&github.User{
		ID: github.Int64(githubMinderID),
	}, nil)
	handler, err := newCommitStatusPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`+    "mongodb": {
+      "version": "5.1.0",
+      }`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	dep := &pb.PrDependencies_ContextualDependency{
		Dep: &pb.Dependency{
			Ecosystem: pb.DepEcosystem_DEP_ECOSYSTEM_NPM,
			Name:      "mongodb",
			Version:   "0.5.0",
		},
		File: &pb.PrDependencies_ContextualDependency_FilePatch{
			Name:     "package-lock.json",
			PatchUrl: server.URL,
		},
	}

	patchPackage := &packageJson{
		Name:    "mongodb",
		Version: "0.6.0",
		Dist: struct {
			Integrity string `json:"integrity"`
			Tarball   string `json:"tarball"`
		}{
			Integrity: "sha512-+1+2+3+4+5+6+7+8+9+0",
			Tarball:   "https://registry.npmjs.org/mongodb/-/mongodb-0.6.0.tgz",
		},
	}
	mockClient.EXPECT().
		NewRequest("GET", server.URL, nil).
		Return(http.NewRequest("GET", server.URL, nil))
	mockClient.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`"%s": {`, patchPackage.Name))),
		}, nil)

	err = handler.trackVulnerableDep(context.TODO(), dep, nil, patchPackage)
	require.NoError(t, err)

	expBody, err := createReviewBody(vulnsFoundText)
	require.NoError(t, err)
	expCommentBody := reviewBodyWithSuggestion(patchPackage.IndentedString(0, fmt.Sprintf(`"%s": {`, patchPackage.Name), nil))

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("COMMENT"),
			Body:     github.String(expBody),
			Comments: []*github.DraftReviewComment{
				{
					Path:      github.String(dep.File.Name),
					StartLine: github.Int(1),
					Line:      github.Int(4),
					Body:      github.String(expCommentBody),
				},
			},
		})

	mockClient.EXPECT().SetCommitStatus(gomock.Any(), pr.RepoOwner, pr.RepoName, commitSHA, &github.RepoStatus{
		State:       github.String("failure"),
		Description: github.String(vulnsFoundTextShort),
		Context:     github.String(commitStatusContext),
	})

	err = handler.submit(context.Background())
	require.NoError(t, err)
}
