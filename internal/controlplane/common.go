//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"google.golang.org/grpc/codes"

	"github.com/stacklok/minder/internal/providers/github/oauth"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	defaultProvider = oauth.Github
	githubURL       = "https://github.com"
)

var validRepoSlugRe = regexp.MustCompile(`(?i)^[-a-z0-9_\.]+\/[-a-z0-9_\.]+$`)

// HasProtoContext is an interface that can be implemented by a request
type HasProtoContext interface {
	GetContext() *pb.Context
}

// providerError wraps an error with a user visible error message
func providerError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return util.UserVisibleError(codes.NotFound, "provider not found")
	}
	return fmt.Errorf("provider error: %w", err)
}

// getNameFilterParam allows us to build a name filter for our provider queries
func getNameFilterParam(name string) sql.NullString {
	return sql.NullString{
		String: name,
		Valid:  name != "",
	}
}

// getRemediationURLFromMetadata returns the "remediation URL". For now, this is
// the URL link to the PR
func getRemediationURLFromMetadata(data []byte, repoSlug string) (string, error) {
	if !validRepoSlugRe.MatchString(repoSlug) {
		return "", fmt.Errorf("invalid repository slug")
	}

	// If no data, it means no PR is tracked.
	// So no error and we return an empty string.
	if len(data) == 0 {
		return "", nil
	}

	prData := &struct {
		Number int `json:"pr_number"`
	}{}

	if err := json.Unmarshal(data, prData); err != nil {
		return "", fmt.Errorf("unmarshalling pull request metadata: %w", err)
	}

	// No pull request found
	if prData.Number == 0 {
		return "", nil
	}

	return fmt.Sprintf("%s/%s/pull/%d", githubURL, repoSlug, prData.Number), nil
}

// getAlertURLFromMetadata is a helper function to get the alert URL from the alert metadata
func getAlertURLFromMetadata(data []byte, repoSlug string) (string, error) {
	if !validRepoSlugRe.MatchString(repoSlug) {
		return "", fmt.Errorf("invalid repository slug")
	}

	// If there is no metadata, we know there is no alert
	if len(data) == 0 {
		return "", nil
	}

	// Define a struct to match the JSON structure
	jsonMeta := struct {
		GhsaId string `json:"ghsa_id"`
	}{}

	if err := json.Unmarshal(data, &jsonMeta); err != nil {
		return "", err
	}

	if jsonMeta.GhsaId == "" {
		return "", nil
	}

	return fmt.Sprintf(
		"%s/%s/security/advisories/%s", githubURL, repoSlug, jsonMeta.GhsaId,
	), nil
}
