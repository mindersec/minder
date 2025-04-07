// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package domain contains the domain logic for the homoglyphs rule type
package domain

import (
	"fmt"

	"github.com/mindersec/minder/internal/engine/eval/homoglyphs/domain/resources"
	"github.com/mindersec/minder/internal/engine/eval/homoglyphs/util"
)

// InvisibleCharactersProcessor is a processor for the invisible characters rule type
type InvisibleCharactersProcessor struct {
	invisibleCharacters map[rune]struct{}
}

// NewInvisibleCharactersProcessor creates a new InvisibleCharactersProcessor
func NewInvisibleCharactersProcessor() HomoglyphProcessor {
	return &InvisibleCharactersProcessor{
		invisibleCharacters: resources.InvisibleCharacters,
	}
}

// FindViolations finds invisible characters in the given line
func (ice *InvisibleCharactersProcessor) FindViolations(line string) []*Violation {
	return ice.FindInvisibleCharacters(line)
}

// GetSubCommentText returns the sub comment text for invisible characters
func (*InvisibleCharactersProcessor) GetSubCommentText() string {
	return "**Invisible Characters Found:**\n\n"
}

// GetLineCommentText returns the line comment text for invisible characters
func (*InvisibleCharactersProcessor) GetLineCommentText(violation *Violation) string {
	if violation == nil {
		return ""
	}

	return fmt.Sprintf("- `%U` \n", violation.InvisibleChar)
}

// GetFailedReviewText returns the failed review text for invisible characters
func (*InvisibleCharactersProcessor) GetFailedReviewText() string {
	return util.InvisibleCharsFoundText
}

// GetPassedReviewText returns the passed review text for invisible characters
func (*InvisibleCharactersProcessor) GetPassedReviewText() string {
	return util.NoInvisibleCharsFoundText
}

// FindInvisibleCharacters checks for invisible characters in the given line
// and returns a slice of runes representing the invisible characters found.
func (ice *InvisibleCharactersProcessor) FindInvisibleCharacters(line string) []*Violation {
	invisibleChars := make([]*Violation, 0)
	for _, r := range line {
		if _, exists := ice.invisibleCharacters[r]; exists {
			invisibleChars = append(invisibleChars, &Violation{InvisibleChar: r})
		}
	}

	return invisibleChars
}
