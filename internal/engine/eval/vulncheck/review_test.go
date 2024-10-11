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

	"github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	pbinternal "github.com/stacklok/minder/internal/proto"
	mock_ghclient "github.com/stacklok/minder/internal/providers/github/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	githubSubmitterID = 144222806
	githubMinderID    = 123456789

	minderReviewID  = 987654321
	minderReviewUrl = "https://github.com/repos/jakubtestorg/bad-npm/pulls/43#pullrequestreview-987654321"

	commitSHA        = "27d6810b861c81e8c61e09c651875f5a976781d1"
	anotherCommitSha = "27d6810b861c81e8c61e09c651875f5a976781d2"
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

	report := createStatusReport(noVulsFoundText, commitSHA, 0)

	expBody, err := report.render()
	require.NoError(t, err)
	require.NotNil(t, expBody)

	mockClient.EXPECT().
		ListIssueComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.IssueComment{}, nil)

	mockClient.EXPECT().
		CreateIssueComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return(&github.IssueComment{ID: github.Int64(123)}, nil)

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

	dep := &pbinternal.PrDependencies_ContextualDependency{
		Dep: &pbinternal.Dependency{
			Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM,
			Name:      "mongodb",
			Version:   "0.5.0",
		},
		File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
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

	statusReport := createStatusReport(vulnsFoundText, commitSHA, 0, dependencyVulnerabilities{
		Dependency:      dep.Dep,
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "0.6.0",
	},
	)

	expStatusBody, err := statusReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expStatusBody)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("REQUEST_CHANGES"),
			Comments: []*github.DraftReviewComment{
				{
					Path:      github.String(dep.File.Name),
					StartLine: github.Int(1),
					Line:      github.Int(4),
					Body:      github.String("```suggestion\n\n  \"version\": \"0.6.0\",\n  \"resolved\": \"https://registry.npmjs.org/mongodb/-/mongodb-0.6.0.tgz\",\n  \"integrity\": \"sha512-+1+2+3+4+5+6+7+8+9+0\",\n```\n"),
				},
			},
		}).Return(&github.PullRequestReview{HTMLURL: github.String(minderReviewUrl)}, nil)

	mockClient.EXPECT().
		ListIssueComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.IssueComment{{
			ID: github.Int64(12345), User: &github.User{ID: github.Int64(githubMinderID)}, Body: github.String(statusBodyMagicComment)},
		}, nil)

	mockClient.EXPECT().
		UpdateIssueComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int64(12345), expStatusBody).
		Return(nil)

	err = handler.submit(context.Background())
	require.NoError(t, err)
}

func TestReviewPrHandlerVulnerabilitiesErrLookUpPackage(t *testing.T) {
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

	dep := &pbinternal.PrDependencies_ContextualDependency{
		Dep: &pbinternal.Dependency{
			Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM,
			Name:      "mongodb",
			Version:   "0.5.0",
		},
		File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
			Name:     "package-lock.json",
			PatchUrl: server.URL,
		},
	}

	patchPackage := &packageJson{
		formatterMeta: formatterMeta{
			pkgRegistryLookupError: ErrPkgNotFound,
		},
		Name:    "mongodb",
		Version: "0.6.0",
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
			{ID: "mongodb"},
		},
	}
	err = handler.trackVulnerableDep(context.TODO(), dep, &vulnResp, patchPackage)
	require.NoError(t, err)

	mockClient.EXPECT().
		ListIssueComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.IssueComment{}, nil)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("REQUEST_CHANGES"),
			Comments: []*github.DraftReviewComment{
				{
					Path: github.String(dep.File.Name),
					Line: github.Int(1),
					Body: github.String(pkgRepoInfoNotFound),
				},
			}}).Return(
		&github.PullRequestReview{
			HTMLURL: github.String(minderReviewUrl),
			ID:      github.Int64(minderReviewID),
		}, nil)

	mockClient.EXPECT().
		CreateIssueComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return(&github.IssueComment{ID: github.Int64(123)}, nil)

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

	dep := &pbinternal.PrDependencies_ContextualDependency{
		Dep: &pbinternal.Dependency{
			Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM,
			Name:      "mongodb",
			Version:   "0.5.0",
		},
		File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
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

	statusReport := createStatusReport(vulnsFoundText, commitSHA, 0, dependencyVulnerabilities{
		Dependency:      dep.Dep,
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "",
	},
	)

	expStatusBody, err := statusReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expStatusBody)

	expCommentBody := fmt.Sprintf(vulnFoundWithNoPatchFmt, dep.Dep.Name)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("REQUEST_CHANGES"),
			Comments: []*github.DraftReviewComment{
				{
					Path: github.String(dep.File.Name),
					Line: github.Int(1),
					Body: github.String(expCommentBody),
				},
			},
		}).Return(&github.PullRequestReview{HTMLURL: github.String(minderReviewUrl)}, nil)

	mockClient.EXPECT().
		ListIssueComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.IssueComment{{
			ID: github.Int64(12345), User: &github.User{ID: github.Int64(githubMinderID)}, Body: github.String(statusBodyMagicComment)},
		}, nil)

	mockClient.EXPECT().
		UpdateIssueComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int64(12345), expStatusBody).
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

	mockClient.EXPECT().GetUserId(gomock.Any()).Return(int64(minderReviewID), nil)
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

	dep := &pbinternal.PrDependencies_ContextualDependency{
		Dep: &pbinternal.Dependency{
			Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM,
			Name:      "mongodb",
			Version:   "0.5.0",
		},
		File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
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

	statusReport := createStatusReport(vulnsFoundText, commitSHA, minderReviewID, dependencyVulnerabilities{
		Dependency:      dep.GetDep(),
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "",
	})

	statusComment, err := render(minderTemplateMagicCommentName, statusBodyMagicComment, magicCommentInfo{
		ContentSha: anotherCommitSha,
		ReviewID:   minderReviewID,
	})
	require.NoError(t, err)

	expStatusBody, err := statusReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expStatusBody)

	mockClient.EXPECT().DismissReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), int64(minderReviewID),
		&github.PullRequestReviewDismissalRequest{
			Message: github.String(reviewBodyDismissCommentText),
		})

	expCommentBody := fmt.Sprintf(vulnFoundWithNoPatchFmt, dep.Dep.Name)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("REQUEST_CHANGES"),
			Comments: []*github.DraftReviewComment{
				{
					Path: github.String(dep.File.Name),
					Line: github.Int(1),
					Body: github.String(expCommentBody),
				},
			}}).Return(
		&github.PullRequestReview{
			HTMLURL: github.String(minderReviewUrl),
			ID:      github.Int64(minderReviewID),
		}, nil)

	mockClient.EXPECT().
		ListIssueComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.IssueComment{{
			ID: github.Int64(12345), User: &github.User{ID: github.Int64(minderReviewID)}, Body: github.String(statusComment)},
		}, nil)

	mockClient.EXPECT().
		UpdateIssueComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int64(12345), expStatusBody).
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

	report := createStatusReport(noVulsFoundText, commitSHA, 0)

	expBody, err := report.render()
	require.NoError(t, err)
	require.NotNil(t, expBody)

	mockClient.EXPECT().
		ListIssueComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.IssueComment{}, nil)

	mockClient.EXPECT().
		CreateIssueComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), expBody).
		Return(&github.IssueComment{ID: github.Int64(123)}, nil)

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

	dep := &pbinternal.PrDependencies_ContextualDependency{
		Dep: &pbinternal.Dependency{
			Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM,
			Name:      "mongodb",
			Version:   "0.5.0",
		},
		File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
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

	statusReport := createStatusReport(vulnsFoundText, commitSHA, 0, dependencyVulnerabilities{
		Dependency:      dep.GetDep(),
		Vulnerabilities: vulnResp.Vulns,
		PatchVersion:    "0.6.0",
	})

	expStatusBody, err := statusReport.render()
	require.NoError(t, err)
	require.NotEmpty(t, expStatusBody)

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), &github.PullRequestReviewRequest{
			CommitID: github.String(commitSHA),
			Event:    github.String("COMMENT"),
			Comments: []*github.DraftReviewComment{
				{
					Path:      github.String(dep.File.Name),
					StartLine: github.Int(1),
					Line:      github.Int(4),
					Body: github.String("```suggestion\n\n" +
						"  \"version\": \"0.6.0\",\n" +
						"  \"resolved\": \"https://registry.npmjs.org/mongodb/-/mongodb-0.6.0.tgz\",\n" +
						"  \"integrity\": \"sha512-+1+2+3+4+5+6+7+8+9+0\",\n" +
						"```\n"),
				},
			},
		}).Return(&github.PullRequestReview{HTMLURL: github.String(minderReviewUrl)}, nil)

	mockClient.EXPECT().
		ListIssueComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.IssueComment{}, nil)

	mockClient.EXPECT().
		CreateIssueComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), expStatusBody).
		Return(&github.IssueComment{ID: github.Int64(123)}, nil)

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

	mockClient.EXPECT().GetUserId(gomock.Any()).Return(int64(githubMinderID), nil)
	handler, err := newReviewPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	mockClient.EXPECT().DismissReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), int64(minderReviewID),
		&github.PullRequestReviewDismissalRequest{
			Message: github.String(reviewBodyDismissCommentText),
		})

	statusComment, err := render(minderTemplateMagicCommentName, statusBodyMagicComment, magicCommentInfo{
		ContentSha: anotherCommitSha,
		ReviewID:   minderReviewID,
	})
	require.NoError(t, err)

	mockClient.EXPECT().
		ListIssueComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.IssueComment{{
			ID: github.Int64(12345), User: &github.User{ID: github.Int64(githubMinderID)}, Body: github.String(statusComment)},
		}, nil)

	mockClient.EXPECT().
		UpdateIssueComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int64(12345), gomock.Any()).
		Return(nil)

	err = handler.submit(context.Background())
	require.NoError(t, err)
	//require.Equal(t, handler., latestReviewOnPR)
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

	mockClient.EXPECT().GetUserId(gomock.Any()).Return(int64(githubMinderID), nil)
	handler, err := newReviewPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// Create a single comment to pretend some vulns were found
	handler.comments = []*github.DraftReviewComment{{
		Body: github.String("test"),
	}}

	mockClient.EXPECT().
		CreateReview(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return(&github.PullRequestReview{HTMLURL: github.String(minderReviewUrl)}, nil)

	mockClient.EXPECT().
		ListIssueComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.IssueComment{{
			ID: github.Int64(12345), User: &github.User{ID: github.Int64(githubMinderID)}, Body: github.String(statusBodyMagicComment)},
		}, nil)

	mockClient.EXPECT().
		UpdateIssueComment(gomock.Any(), pr.RepoOwner, pr.RepoName, int64(12345), gomock.Any()).
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

	mockClient.EXPECT().GetUserId(gomock.Any()).Return(int64(githubMinderID), nil)
	handler, err := newReviewPrHandler(context.TODO(), pr, mockClient)
	require.NoError(t, err)
	require.NotNil(t, handler)

	statusComment, err := render(minderTemplateMagicCommentName, statusBodyMagicComment, magicCommentInfo{
		ContentSha: commitSHA,
		ReviewID:   minderReviewID,
	})
	require.NoError(t, err)

	mockClient.EXPECT().
		ListIssueComments(gomock.Any(), pr.RepoOwner, pr.RepoName, int(pr.Number), gomock.Any()).
		Return([]*github.IssueComment{{
			ID:   github.Int64(12345),
			User: &github.User{ID: github.Int64(githubMinderID)},
			Body: github.String(statusComment),
		},
		}, nil)

	err = handler.submit(context.Background())
	require.NoError(t, err)
	require.Contains(t, handler.minderStatusReport.GetBody(), commitSHA)
}

//nolint:unparam
func createStatusReport(reviewText, sha string, reviewID int64, deps ...dependencyVulnerabilities) vulnerabilityReport {
	return &statusReport{
		StatusText:          reviewText,
		TrackedDependencies: deps,
		CommitSHA:           sha,
		ReviewID:            reviewID,
	}
}
