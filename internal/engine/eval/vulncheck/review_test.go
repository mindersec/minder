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

	minderReviewID  = 987654321
	minderReviewUrl = "https://github.com/repos/jakubtestorg/bad-npm/pulls/43#pullrequestreview-987654321"

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

	mockClient.EXPECT().GetUserId(gomock.Any()).Return(int64(githubSubmitterID), nil)
	handler, err := newReviewPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	report := createStatusReport(noVulsFoundText, commitSHA)

	expBody, err := report.render()
	require.NoError(t, err)
	require.NotNil(t, expBody)

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		ListComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.PullRequestComment{}, nil)

	mockClient.EXPECT().
		CreateComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return(&github.PullRequestComment{ID: github.Int64(123)}, nil)

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

	mockClient.EXPECT().GetUserId(gomock.Any()).Return(int64(githubMinderID), nil)
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

	vulnResp := VulnerabilityResponse{
		[]Vulnerability{
			{ID: "mongodb", Fixed: "0.6.0"},
		},
	}
	err = handler.trackVulnerableDep(context.TODO(), dep, &vulnResp, patchPackage)
	require.NoError(t, err)

	reviewReport := createReviewReport(dependencyVulnerabilities{
		Dependency:      dep.Dep,
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "0.6.0",
	},
	)

	statusReport := createStatusReport(vulnsFoundText, commitSHA, dependencyVulnerabilities{
		Dependency:      dep.Dep,
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "0.6.0",
	},
	)

	expReviewBody, err := reviewReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expReviewBody)

	expStatusBody, err := statusReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expStatusBody)

	expCommentBody := reviewBodyWithSuggestion(patchPackage.IndentedString(0, fmt.Sprintf(`"%s": {`, patchPackage.Name), nil))

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("REQUEST_CHANGES"),
			Body:     github.String(expReviewBody),
			Comments: []*github.DraftReviewComment{
				{
					Path:      github.String(dep.File.Name),
					StartLine: github.Int(1),
					Line:      github.Int(4),
					Body:      github.String(expCommentBody),
				},
			},
		}).Return(&github.PullRequestReview{HTMLURL: github.String(minderReviewUrl)}, nil)

	mockClient.EXPECT().
		ListComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.PullRequestComment{{
			ID: github.Int64(12345), User: &github.User{ID: github.Int64(githubMinderID)}, Body: github.String(statusBodyMagicComment)},
		}, nil)

	mockClient.EXPECT().
		UpdateComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int64(12345), expStatusBody).
		Return(nil)

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

	mockClient.EXPECT().GetUserId(gomock.Any()).Return(int64(githubMinderID), nil)
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

	vulnResp := VulnerabilityResponse{
		[]Vulnerability{
			{ID: "mongodb"},
		},
	}
	err = handler.trackVulnerableDep(context.TODO(), dep, &vulnResp, patchPackage)
	require.NoError(t, err)

	reviewReport := createReviewReport(dependencyVulnerabilities{
		Dependency:      dep.Dep,
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "",
	},
	)

	statusReport := createStatusReport(vulnsFoundText, commitSHA, dependencyVulnerabilities{
		Dependency:      dep.Dep,
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "",
	},
	)

	expReviewBody, err := reviewReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expReviewBody)

	expStatusBody, err := statusReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expStatusBody)

	expCommentBody := vulnFoundWithNoPatch

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("REQUEST_CHANGES"),
			Body:     github.String(expReviewBody),
			Comments: []*github.DraftReviewComment{
				{
					Path: github.String(dep.File.Name),
					Line: github.Int(1),
					Body: github.String(expCommentBody),
				},
			},
		}).Return(&github.PullRequestReview{HTMLURL: github.String(minderReviewUrl)}, nil)

	mockClient.EXPECT().
		ListComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.PullRequestComment{{
			ID: github.Int64(12345), User: &github.User{ID: github.Int64(githubMinderID)}, Body: github.String(statusBodyMagicComment)},
		}, nil)

	mockClient.EXPECT().
		UpdateComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int64(12345), expStatusBody).
		Return(nil)

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

<<<<<<< HEAD
	mockClient.EXPECT().GetUserId(gomock.Any()).Return(int64(githubSubmitterID), nil)
=======
	mockClient.EXPECT().GetAuthenticatedUser(gomock.Any()).Return(&github.User{
		ID: github.Int64(githubMinderID),
	}, nil)
>>>>>>> 03dc7218f (Create single status comment and correctly dismiss reviews)
	handler, err := newReviewPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	vulnResp := VulnerabilityResponse{
		[]Vulnerability{
			{ID: "mongodb"},
		},
	}

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

	reviewReport := createReviewReport(dependencyVulnerabilities{
		Dependency:      dep.GetDep(),
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "",
	},
	)
	statusReport := createStatusReport(vulnsFoundText, commitSHA, dependencyVulnerabilities{
		Dependency:      dep.GetDep(),
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "",
	})

	expReviewBody, err := reviewReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expReviewBody)

	expStatusBody, err := statusReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expStatusBody)

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{
			{
				ID:   github.Int64(minderReviewID),
				Body: github.String(reviewBodyMagicComment),
				User: &github.User{ID: github.Int64(githubMinderID)},
			},
		}, nil)

	mockClient.EXPECT().DismissReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), int64(minderReviewID),
		&github.PullRequestReviewDismissalRequest{
			Message: github.String(reviewBodyDismissCommentText),
		})

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("REQUEST_CHANGES"),
			Body:     github.String(expReviewBody),
			Comments: []*github.DraftReviewComment{
				{
					Path: github.String(dep.File.Name),
					Line: github.Int(1),
					Body: github.String(vulnFoundWithNoPatch),
				},
			}}).Return(&github.PullRequestReview{HTMLURL: github.String(minderReviewUrl)}, nil)

	mockClient.EXPECT().
		ListComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.PullRequestComment{{
			ID: github.Int64(12345), User: &github.User{ID: github.Int64(githubMinderID)}, Body: github.String(statusBodyMagicComment)},
		}, nil)

	mockClient.EXPECT().
		UpdateComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int64(12345), expStatusBody).
		Return(nil)

	err = handler.trackVulnerableDep(context.TODO(), dep, &vulnResp, patchPackage)
	require.NoError(t, err)

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

	mockClient.EXPECT().GetUserId(gomock.Any()).Return(int64(githubSubmitterID), nil)
	handler, err := newCommitStatusPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	report := createStatusReport(noVulsFoundText, commitSHA)

	expBody, err := report.render()
	require.NoError(t, err)
	require.NotNil(t, expBody)

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		ListComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.PullRequestComment{}, nil)

	mockClient.EXPECT().
		CreateComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), expBody).
		Return(&github.PullRequestComment{ID: github.Int64(123)}, nil)

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

	mockClient.EXPECT().GetUserId(gomock.Any()).Return(int64(githubMinderID), nil)
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

	vulnResp := VulnerabilityResponse{
		[]Vulnerability{
			{ID: "mongodb", Fixed: "0.6.0"},
		},
	}
	err = handler.trackVulnerableDep(context.TODO(), dep, &vulnResp, patchPackage)
	require.NoError(t, err)

	reviewReport := createReviewReport(dependencyVulnerabilities{
		Dependency:      dep.GetDep(),
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "0.6.0",
	},
	)
	statusReport := createStatusReport(vulnsFoundText, commitSHA, dependencyVulnerabilities{
		Dependency:      dep.GetDep(),
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "0.6.0",
	})

	expReviewBody, err := reviewReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expReviewBody)

	expStatusBody, err := statusReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expStatusBody)

	expCommentBody := reviewBodyWithSuggestion(patchPackage.IndentedString(0, fmt.Sprintf(`"%s": {`, patchPackage.Name), nil))

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("COMMENT"),
			Body:     github.String(expReviewBody),
			Comments: []*github.DraftReviewComment{
				{
					Path:      github.String(dep.File.Name),
					StartLine: github.Int(1),
					Line:      github.Int(4),
					Body:      github.String(expCommentBody),
				},
			},
		}).Return(&github.PullRequestReview{HTMLURL: github.String(minderReviewUrl)}, nil)

	mockClient.EXPECT().
		ListComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.PullRequestComment{}, nil)

	mockClient.EXPECT().
		CreateComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), expStatusBody).
		Return(&github.PullRequestComment{ID: github.Int64(123)}, nil)

	mockClient.EXPECT().SetCommitStatus(gomock.Any(), pr.RepoOwner, pr.RepoName, commitSHA, &github.RepoStatus{
		State:       github.String("failure"),
		Description: github.String(vulnsFoundTextShort),
		Context:     github.String(commitStatusContext),
	})

	err = handler.submit(context.Background())
	require.NoError(t, err)
}

func TestReviewPrHandlerReviewPriorReview(t *testing.T) {
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

	createPrReviewWithSHA := func(review, sha string, user int64) *github.PullRequestReview {
		return &github.PullRequestReview{
			ID:       github.Int64(minderReviewID),
			Body:     github.String(review),
			CommitID: github.String(sha),
			State:    github.String("COMMENTED"),
			User:     &github.User{ID: github.Int64(user)},
		}
	}

	latestReviewOnPR := createPrReviewWithSHA(reviewBodyMagicComment, "latestCommitSHA", githubMinderID)
	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{
			createPrReviewWithSHA(statusBodyMagicComment, "94ea8a313784de6d65dccbe0fc815335bf417633", githubMinderID),
			createPrReviewWithSHA("test-review", "0105b620f68582a3e20da211c20d72798ef40d77", 12345),
			createPrReviewWithSHA("test-review", "3410b66b103cbf1ed777462c7ebba8b249b74dce", 54321),
			createPrReviewWithSHA(reviewBodyMagicComment, "277a82af28ffdbe3ab3c1209e27ddcc0774d3da7", githubMinderID),
			createPrReviewWithSHA("test-review", "5e26642d64aec7bae43167d8ad20cb090d33141a", 56789),
			latestReviewOnPR, // review for latest SHA on PR
		}, nil)

	mockClient.EXPECT().DismissReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), int64(minderReviewID),
		&github.PullRequestReviewDismissalRequest{
			Message: github.String(reviewBodyDismissCommentText),
		})

	mockClient.EXPECT().
		ListComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.PullRequestComment{{
			ID: github.Int64(12345), User: &github.User{ID: github.Int64(githubMinderID)}, Body: github.String(statusBodyMagicComment)},
		}, nil)

	mockClient.EXPECT().
		UpdateComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int64(12345), gomock.Any()).
		Return(nil)

	err = handler.submit(context.Background())
	require.NoError(t, err)
	require.Equal(t, handler.minderReview, latestReviewOnPR)
}

func TestReviewPrHandlerVulnerabilitiesAndNoPriorReview(t *testing.T) {
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

	// Create a single comment to pretend some vulns were found
	handler.comments = []*github.DraftReviewComment{{
		Body: github.String("test"),
	}}

	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{}, nil)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return(&github.PullRequestReview{HTMLURL: github.String(minderReviewUrl)}, nil)

	mockClient.EXPECT().
		ListComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.PullRequestComment{{
			ID: github.Int64(12345), User: &github.User{ID: github.Int64(githubMinderID)}, Body: github.String(statusBodyMagicComment)},
		}, nil)

	mockClient.EXPECT().
		UpdateComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int64(12345), gomock.Any()).
		Return(nil)

	err = handler.submit(context.Background())
	require.NoError(t, err)
}

func TestReviewPrHandlerReviewAlreadyExistsOnSHA(t *testing.T) {
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

	createPrReviewWithSHA := func(review, sha string, user int64) *github.PullRequestReview {
		return &github.PullRequestReview{
			ID:       github.Int64(minderReviewID),
			Body:     github.String(review),
			CommitID: github.String(sha),
			State:    github.String("COMMENTED"),
			User:     &github.User{ID: github.Int64(user)},
		}
	}

	latestReviewOnPR := createPrReviewWithSHA(reviewBodyMagicComment, commitSHA, githubMinderID)
	mockClient.EXPECT().
		ListReviews(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), nil).
		Return([]*github.PullRequestReview{
			createPrReviewWithSHA(statusBodyMagicComment, "94ea8a313784de6d65dccbe0fc815335bf417633", githubMinderID),
			createPrReviewWithSHA("test-review", "0105b620f68582a3e20da211c20d72798ef40d77", 12345),
			createPrReviewWithSHA("test-review", "3410b66b103cbf1ed777462c7ebba8b249b74dce", 54321),
			createPrReviewWithSHA(reviewBodyMagicComment, "277a82af28ffdbe3ab3c1209e27ddcc0774d3da7", githubMinderID),
			createPrReviewWithSHA("test-review", "5e26642d64aec7bae43167d8ad20cb090d33141a", 56789),
			latestReviewOnPR, // review for latest SHA on PR
		}, nil)

	mockClient.EXPECT().
		ListComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.PullRequestComment{{
			ID:       github.Int64(12345),
			User:     &github.User{ID: github.Int64(githubMinderID)},
			Body:     github.String(statusBodyMagicComment),
			CommitID: github.String(commitSHA),
		},
		}, nil)

	err = handler.submit(context.Background())
	require.NoError(t, err)
	require.Equal(t, handler.minderReview, latestReviewOnPR)
}

//nolint:unparam
func createStatusReport(reviewText, sha string, deps ...dependencyVulnerabilities) vulnerabilityReport {
	return &statusReport{
		StatusText:          reviewText,
		TrackedDependencies: deps,
		CommitSHA:           sha,
	}
}

func createReviewReport(deps ...dependencyVulnerabilities) vulnerabilityReport {
	return &reviewReport{
		TrackedDependencies: deps,
	}
}
