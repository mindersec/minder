package domain

// HomoglyphProcessor is an interface for a homoglyph processor
type HomoglyphProcessor interface {
	FindViolations(line string) []*Violation
	GetSubCommentText() string
	GetLineCommentText(violation *Violation) string
	GetPassedReviewText() string
	GetFailedReviewText() string
}
