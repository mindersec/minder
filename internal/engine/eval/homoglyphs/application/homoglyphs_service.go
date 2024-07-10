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

// Package application contains the application logic for the homoglyphs rule type
package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v61/github"

	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/communication"
	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/domain"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/engine/models"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// HomoglyphsEvalType is the type of the homoglyphs evaluator
	HomoglyphsEvalType = "homoglyphs"

	invisibleCharacters = "invisible_characters"
	mixedScript         = "mixed_scripts"
)

// NewHomoglyphsEvaluator creates a new homoglyphs evaluator
func NewHomoglyphsEvaluator(
	reh *pb.RuleType_Definition_Eval_Homoglyphs,
	ghClient provifv1.GitHub,
) (engif.Evaluator, error) {
	if ghClient == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}
	if reh == nil {
		return nil, fmt.Errorf("homoglyphs configuration is nil")
	}

	switch reh.Type {
	case invisibleCharacters:
		return NewInvisibleCharactersEvaluator(ghClient)
	case mixedScript:
		return NewMixedScriptEvaluator(ghClient)
	default:
		return nil, fmt.Errorf("unsupported homoglyphs type: %s", reh.Type)
	}
}

// evaluateHomoglyphs is a helper function to evaluate the homoglyphs rule type
// Return parameters:
// - bool: whether the evaluation has found violations
// - error: an error if the evaluation failed
func evaluateHomoglyphs(
	ctx context.Context,
	processor domain.HomoglyphProcessor,
	res *engif.Result,
	reviewHandler *communication.GhReviewPrHandler,
) (bool, error) {
	if res == nil {
		return false, fmt.Errorf("result is nil")
	}

	//nolint:govet
	prContents, ok := res.Object.(*models.PRContents)
	if !ok {
		return false, fmt.Errorf("invalid object type for homoglyphs evaluator")
	}

	if prContents.PR == nil || prContents.Files == nil {
		return false, fmt.Errorf("invalid prContents fields: %v, %v", prContents.PR, prContents.Files)
	}

	if len(prContents.Files) == 0 {
		return false, nil
	}

	// Note: This is a mandatory step to reassign certain fields in the handler.
	// This is a workaround to avoid recreating the object.
	reviewHandler.Hydrate(ctx, prContents.PR)

	for _, file := range prContents.Files {
		for _, line := range file.PatchLines {
			violations := processor.FindViolations(line.Content)
			if len(violations) == 0 {
				continue
			}

			var commentBody strings.Builder
			commentBody.WriteString(processor.GetSubCommentText())

			for _, v := range violations {
				commentBody.WriteString(processor.GetLineCommentText(v))
			}

			reviewComment := &github.DraftReviewComment{
				Path: github.String(file.Name),
				Body: github.String(commentBody.String()),
				Line: github.Int(int(line.LineNumber)),
			}

			reviewHandler.AddComment(reviewComment)
		}
	}

	if len(reviewHandler.GetComments()) > 0 {
		return true, reviewHandler.SubmitReview(ctx, processor.GetFailedReviewText())
	}

	return false, nil
}
