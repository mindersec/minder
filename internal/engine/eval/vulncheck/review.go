// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
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
	"strings"

	"github.com/google/go-github/v53/github"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

type reviewLocation struct {
	lineToChange      int
	leadingWhitespace int
}

func countLeadingWhitespace(line string) int {
	count := 0
	for _, ch := range line {
		if ch != ' ' && ch != '\t' {
			return count
		}
		count++
	}
	return count
}

func locateDepInPr(
	_ context.Context,
	client ghclient.RestAPI,
	dep *pb.PrDependencies_ContextualDependency,
) (*reviewLocation, error) {
	req, err := client.NewRequest("GET", dep.File.PatchUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}
	// TODO:(jakub) I couldn't make this work with the GH client
	netClient := &http.Client{}
	resp, _ := netClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	loc := reviewLocation{}
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		pkgName := fmt.Sprintf(`"%s": {`, dep.Dep.Name)
		if strings.Contains(line, pkgName) {
			loc.leadingWhitespace = countLeadingWhitespace(line)
			loc.lineToChange = i + 1
		}
	}

	return &loc, nil
}

func requestChanges(
	ctx context.Context,
	client ghclient.RestAPI,
	fileName string,
	pr *pb.PullRequest,
	location *reviewLocation,
	comment string,
) error {
	var comments []*github.DraftReviewComment

	body := fmt.Sprintf("```suggestion\n"+"%s\n"+"```\n", comment)

	reviewComment := &github.DraftReviewComment{
		Path:      github.String(fileName),
		Position:  nil,
		StartLine: github.Int(location.lineToChange),
		Line:      github.Int(location.lineToChange + 3), // TODO(jakub): Need to count the lines from the patch
		Body:      github.String(body),
	}
	comments = append(comments, reviewComment)

	review := &github.PullRequestReviewRequest{
		CommitID: github.String(pr.CommitSha),
		Event:    github.String("REQUEST_CHANGES"),
		Comments: comments,
	}

	_, err := client.CreateReview(
		ctx,
		pr.RepoOwner,
		pr.RepoName,
		int(pr.Number),
		review,
	)
	if err != nil {
		return fmt.Errorf("could not create review: %w", err)
	}

	return nil
}
