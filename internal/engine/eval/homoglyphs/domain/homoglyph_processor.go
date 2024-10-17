// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package domain

// HomoglyphProcessor is an interface for a homoglyph processor
type HomoglyphProcessor interface {
	FindViolations(line string) []*Violation
	GetSubCommentText() string
	GetLineCommentText(violation *Violation) string
	GetPassedReviewText() string
	GetFailedReviewText() string
}
