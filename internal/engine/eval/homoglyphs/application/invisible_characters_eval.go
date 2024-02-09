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

package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v56/github"

	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/communication"
	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/domain"
	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/util"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// InvisibleCharactersEvaluator is an evaluator for the invisible characters rule type
type InvisibleCharactersEvaluator struct {
	processor     *domain.InvisibleCharactersProcessor
	reviewHandler *communication.GhReviewPrHandler
}

// NewInvisibleCharactersEvaluator creates a new invisible characters evaluator
func NewInvisibleCharactersEvaluator(pbuild *providers.ProviderBuilder) (*InvisibleCharactersEvaluator, error) {
	if pbuild == nil {
		return nil, fmt.Errorf("provider builder is nil")
	}

	ghClient, err := pbuild.GetGitHub(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not fetch GitHub client: %w", err)
	}

	return &InvisibleCharactersEvaluator{
		processor:     domain.NewInvisibleCharactersProcessor(),
		reviewHandler: communication.NewGhReviewPrHandler(ghClient),
	}, nil
}

// Eval evaluates the invisible characters rule type
func (ice *InvisibleCharactersEvaluator) Eval(ctx context.Context, _ map[string]any, res *engif.Result) error {
	if res == nil {
		return fmt.Errorf("result is nil")
	}

	//nolint:govet
	prContents, ok := res.Object.(pb.PrContents)
	if !ok {
		return fmt.Errorf("invalid object type for homoglyphs evaluator")
	}

	if prContents.Pr == nil || prContents.Files == nil {
		return fmt.Errorf("invalid prContents fields: %v, %v", prContents.Pr, prContents.Files)
	}

	if len(prContents.Files) == 0 {
		return nil
	}

	// Note: This is a mandatory step to reassign certain fields in the handler.
	// This is a workaround to avoid recreating the object.
	ice.reviewHandler.Hydrate(ctx, prContents.Pr)

	for _, file := range prContents.Files {
		for _, line := range file.PatchLines {
			invisibleCharactersFound := ice.processor.FindInvisibleCharacters(line.Content)
			if len(invisibleCharactersFound) == 0 {
				continue
			}

			var commentBody strings.Builder
			commentBody.WriteString("**Invisible Characters Found:**\n\n")

			for _, r := range invisibleCharactersFound {
				commentBody.WriteString(fmt.Sprintf("- `%U` \n", r))
			}

			reviewComment := &github.DraftReviewComment{
				Path: github.String(file.Name),
				Body: github.String(commentBody.String()),
				Line: github.Int(int(line.LineNumber)),
			}

			ice.reviewHandler.AddComment(reviewComment)
		}
	}

	var reviewText string
	if len(ice.reviewHandler.GetComments()) > 0 {
		reviewText = util.InvisibleCharsFoundText
	} else {
		reviewText = util.NoInvisibleCharsFoundText
	}

	return ice.reviewHandler.SubmitReview(ctx, reviewText)
}
