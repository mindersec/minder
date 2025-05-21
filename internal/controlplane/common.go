// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	defaultProvider = clients.Github
	githubURL       = "https://github.com"
)

var validRepoSlugRe = regexp.MustCompile(`(?i)^[-a-z0-9_\.]+\/[-a-z0-9_\.]+$`)

var (
	// ErrNoProjectInContext is returned when no project is found in the context
	ErrNoProjectInContext = errors.New("no project found in context")
)

// ProviderGetter is an interface that can be implemented by a context,
// since both the context V1 and V2 have a provider field
type ProviderGetter interface {
	GetProvider() string
}

// HasProtoContextV2 is an interface that can be implemented by a request
type HasProtoContextV2 interface {
	GetContext() *pb.ContextV2
}

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

// GetProjectID retrieves the project ID from the request context.
func GetProjectID(ctx context.Context) uuid.UUID {
	entityCtx := engcontext.EntityFromContext(ctx)
	return entityCtx.Project.ID
}

// GetProviderName retrieves the provider name from the request context.
func GetProviderName(ctx context.Context) string {
	entityCtx := engcontext.EntityFromContext(ctx)
	return entityCtx.Provider.Name
}
