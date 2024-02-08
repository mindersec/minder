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
