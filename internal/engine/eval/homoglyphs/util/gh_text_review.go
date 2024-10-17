// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package util contains utility functions for the homoglyphs evaluation engine
package util

import (
	"bytes"
	"text/template"
)

const (
	// ReviewBodyMagicComment is a magic comment that is added to the PR review body
	ReviewBodyMagicComment = "<!-- minder: pr-review-homoglyphs-body -->"
	// ReviewBodyDismissCommentText is a comment that is added to the PR review body when a previous review was dismissed
	ReviewBodyDismissCommentText = "Previous Minder review was dismissed because the PR was updated."

	// ReviewTemplateName is the name of the template used to create the PR review body
	ReviewTemplateName = "reviewHomoglpyhsBody"
	// ReviewTmplStr is the template string used to create the PR review body
	ReviewTmplStr = "{{.MagicComment}}\n\n{{.ReviewText}}"

	// InvisibleCharsFoundText is the text to display when invisible characters are found
	InvisibleCharsFoundText = "### :warning: Minder Has Identified Potential Invisible Unicode Characters\n\n" +
		"These characters could be indicative of malicious code practices.\n" +
		"Please review the content to ensure its integrity and safety."
	// NoInvisibleCharsFoundText is the text to display when no invisible characters are found
	NoInvisibleCharsFoundText = "### :white_check_mark: No Invisible Unicode Characters Detected."

	// MixedScriptsFoundText is the text to display when mixed scripts are found
	MixedScriptsFoundText = "### :warning: Minder Has Identified Potential Mixed Scripts\n\n" +
		"This combination can be indicative of an attempt to obscure malicious code or phishing attempts.\n" +
		"Please review the content carefully to ensure its integrity and safety."
	// NoMixedScriptsFoundText is the text to display when no mixed scripts are found
	NoMixedScriptsFoundText = "### :white_check_mark: No Mixed Scripts Detected."
)

// CreateReviewBody creates a review body for a PR review
func CreateReviewBody(reviewText string) (string, error) {
	tmpl, err := template.New(ReviewTemplateName).Option("missingkey=error").Parse(ReviewTmplStr)
	if err != nil {
		return "", err
	}

	data := &struct {
		MagicComment string
		ReviewText   string
	}{
		MagicComment: ReviewBodyMagicComment,
		ReviewText:   reviewText,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
