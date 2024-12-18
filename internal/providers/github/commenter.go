// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/ptr"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// MagicCommentLimit is the maximum length of the magic comment
	MagicCommentLimit = 1024
	// CommentLimit is the maximum length of the comment
	CommentLimit                   = 65536
	minderTemplateMagicCommentName = "minderCommentBody"
	//nolint:lll
	statusBodyMagicComment       = `<!-- minder: pr-status-body: { "ContentSha": "{{.ContentSha}}", "ReviewID": "{{.ReviewID}}" } -->`
	statusBodyMagicCommentPrefix = "<!-- minder: pr-status-body: "

	minderTemplateName   = "minderCommentBody"
	minderTemplateString = "{{ .MagicComment }}\n\n{{ .Body }}"

	reviewBodyDismissCommentText = "Previous Minder review was dismissed because the PR was updated"
)

type magicCommentInfo struct {
	ContentSha        string `json:"ContentSha"`
	ReviewID          int64  `json:"ReviewID"`
	ExistingCommentID int64
	URL               string
	SubmittedAt       time.Time
}

type minderTemplateData struct {
	MagicComment string
	Body         string
}

func (mci *magicCommentInfo) render(ctx context.Context, body string) (string, error) {
	mcictmpl, err := util.NewSafeTextTemplate(ptr.Ptr(statusBodyMagicComment), minderTemplateMagicCommentName)
	if err != nil {
		return "", fmt.Errorf("could not create magic comment: %w", err)
	}

	magicComment, err := mcictmpl.Render(ctx, mci, MagicCommentLimit)
	if err != nil {
		return "", fmt.Errorf("could not render magic comment: %w", err)
	}

	bodytmpl, err := util.NewSafeTextTemplate(ptr.Ptr(minderTemplateString), minderTemplateName)
	if err != nil {
		return "", fmt.Errorf("could not create body template: %w", err)
	}
	return bodytmpl.Render(ctx, minderTemplateData{
		MagicComment: magicComment,
		Body:         body,
	}, CommentLimit)
}

func (mci *magicCommentInfo) ToCommentResultMeta() *provifv1.CommentResultMeta {
	return &provifv1.CommentResultMeta{
		ID:          strconv.FormatInt(mci.ReviewID, 10),
		URL:         mci.URL,
		SubmittedAt: mci.SubmittedAt,
	}
}

func (c *GitHub) findPreviousStatusComment(
	ctx context.Context, owner, repoName string, prNum int64, authorizedUser int64,
) (magicCommentInfo, error) {
	// TODO: overflow
	comments, _, err := c.client.Issues.ListComments(ctx, owner, repoName, int(prNum),
		&github.IssueListCommentsOptions{
			Sort:      github.String("updated"),
			Direction: github.String("asc"),
		})
	if err != nil {
		return magicCommentInfo{}, fmt.Errorf("could not list comments: %w", err)
	}

	for _, comment := range comments {
		isMinder := comment.GetUser().GetID() == authorizedUser
		if isMinder && strings.HasPrefix(comment.GetBody(), statusBodyMagicCommentPrefix) {
			return getMagicCommentInfo(ctx, comment), nil
		}
	}

	return magicCommentInfo{}, nil
}
func getMagicCommentInfo(
	ctx context.Context, cmt *github.IssueComment,
) magicCommentInfo {
	if cmt == nil {
		return magicCommentInfo{}
	}

	mci, err := extractContentShaAndReviewID(cmt.GetBody())
	if err != nil {
		zerolog.Ctx(ctx).Warn().
			Msg("could not extract content sha and review id from previous status comment")
		// non-fatal error, we can still post the status report
		// and worst case we will add a duplicate comment
	}

	mci.ExistingCommentID = cmt.GetID()
	mci.URL = cmt.GetHTMLURL()
	mci.SubmittedAt = cmt.GetUpdatedAt().Time
	return mci
}

func (c *GitHub) updateOrSubmitComment(ctx context.Context, mci magicCommentInfo,
	comment provifv1.PullRequestCommentInfo,
	owner, repoName string, prNumber int64, commitSha string, prUserID int64, authorizedUserID int64,
) (magicCommentInfo, error) {
	mci, err := c.doReview(
		ctx, comment, mci, owner, repoName, prNumber, commitSha, prUserID, authorizedUserID)
	if err != nil {
		return mci, fmt.Errorf("could not do review: %w", err)
	}

	if mci.ExistingCommentID == 0 {
		// We ensure that the content sha is set
		mci.ContentSha = commitSha
		// Ensures the comment we render has the magic comment info
		body, err := mci.render(ctx, comment.Body)
		if err != nil {
			return mci, fmt.Errorf("could not render comment: %w", err)
		}

		ic, err := c.CreateIssueComment(ctx, owner, repoName,
			// TODO: overflow
			int(prNumber), body)
		if err != nil {
			return mci, fmt.Errorf("failed to create minder status report comment: %w", err)
		}

		mci.URL = ic.GetHTMLURL()
		mci.ReviewID = ic.GetID()
		return mci, nil
	}

	// Ensures the comment we render has the magic comment info
	body, err := mci.render(ctx, comment.Body)
	if err != nil {
		return mci, fmt.Errorf("could not render comment: %w", err)
	}

	// TODO: Should we keep this?
	if mci.ContentSha != comment.Commit {
		err := c.UpdateIssueComment(ctx, owner, repoName, mci.ExistingCommentID, body)
		if err != nil {
			return mci, fmt.Errorf("failed to update minder status report comment: %w", err)
		}

		mci.SubmittedAt = time.Now()
		return mci, nil
	}

	return mci, nil
}

func (c *GitHub) doReview(
	ctx context.Context, comment provifv1.PullRequestCommentInfo, mci magicCommentInfo,
	owner, repoName string, prNumber int64, commitSha string, prUserID int64, authorizedUserID int64,
) (magicCommentInfo, error) {
	logger := zerolog.Ctx(ctx)

	// if the previous review was on the same commit, keep it
	// TODO: This should only apply if the profile has not changed. That is, we need
	// to detect if there was a change in the profile and if there was, we should
	// dismiss the previous review.
	if mci.ContentSha == commitSha {
		logger.Debug().
			Int64("review-id", mci.ReviewID).
			Msg("previous review was on the same commit, will keep it")
		return mci, nil
	}

	if err := c.dismissReview(
		ctx, mci, owner, repoName, prNumber, prUserID, authorizedUserID,
	); err != nil {
		zerolog.Ctx(ctx).Debug().Msg("could not dismiss previous review")
	}

	// We only create a review if the comment is an approval or a request for changes
	if comment.Type == provifv1.PullRequestCommentTypeApprove ||
		comment.Type == provifv1.PullRequestCommentTypeRequestChanges {
		return c.createReview(ctx, comment, mci, owner, repoName, prNumber, commitSha)
	}

	return mci, nil
}

func (c *GitHub) createReview(
	ctx context.Context, comment provifv1.PullRequestCommentInfo, mci magicCommentInfo,
	owner, repoName string, prNumber int64, commitSha string,
) (magicCommentInfo, error) {
	status := "APPROVE"
	body := "Minder has approved this PR"
	if comment.Type == provifv1.PullRequestCommentTypeRequestChanges {
		status = "REQUEST_CHANGES"
		body = "Minder has requested changes"
	}

	review := &github.PullRequestReviewRequest{
		CommitID: github.String(commitSha),
		Event:    github.String(status),
		// TODO: Add comments support
		// Comments: ra.comments
		Body: github.String(body),
	}

	// TODO: overflow
	r, err := c.CreateReview(ctx, owner, repoName, int(prNumber), review)
	if err != nil {
		return mci, fmt.Errorf("could not create review: %w", err)
	}

	mci.ReviewID = r.GetID()
	return mci, nil
}

func (c *GitHub) dismissReview(
	ctx context.Context, mci magicCommentInfo,
	owner, repoName string, prNumber int64,
	prUserID int64, authorizedUserID int64,
) error {
	logger := zerolog.Ctx(ctx)
	if mci.ReviewID == 0 {
		logger.Debug().Msg("no previous review to dismiss")
		return nil
	}

	if prUserID == authorizedUserID {
		logger.Warn().Msg("author is the same as the authenticated user, can't dismiss")
		return nil
	}

	dismissReview := &github.PullRequestReviewDismissalRequest{
		Message: github.String(reviewBodyDismissCommentText),
	}

	// TODO: overflow
	_, err := c.DismissReview(ctx, owner, repoName, int(prNumber), mci.ReviewID, dismissReview)
	if err != nil {
		return fmt.Errorf("could not dismiss review: %w", err)
	}
	return nil
}

func extractContentShaAndReviewID(input string) (magicCommentInfo, error) {
	re := regexp.MustCompile(fmt.Sprintf("%s(\\{.*?\\}) -->", statusBodyMagicCommentPrefix))

	matches := re.FindStringSubmatch(input)
	if len(matches) != 2 {
		return magicCommentInfo{}, errors.New("no match found")
	}

	jsonPart := matches[1]

	var strMagicCommentInfo struct {
		ContentSha string `json:"ContentSha"`
		ReviewID   string `json:"ReviewID"` // Assuming you're handling ReviewID as a string
	}
	err := json.Unmarshal([]byte(jsonPart), &strMagicCommentInfo)
	if err != nil {
		return magicCommentInfo{}, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	var contentInfo magicCommentInfo
	contentInfo.ContentSha = strMagicCommentInfo.ContentSha
	contentInfo.ReviewID, err = strconv.ParseInt(strMagicCommentInfo.ReviewID, 10, 64)
	if err != nil {
		return magicCommentInfo{}, fmt.Errorf("error parsing ReviewID: %w", err)
	}

	return contentInfo, nil
}
