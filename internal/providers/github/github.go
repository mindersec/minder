// Copyright 2023 Stacklok, Inc
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

// Package github provides a client for interacting with the GitHub API
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	backoffv4 "github.com/cenkalti/backoff/v4"
	"github.com/google/go-github/v56/github"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// ExpensiveRestCallTimeout is the timeout for expensive REST calls
	ExpensiveRestCallTimeout = 15 * time.Second
	// MaxRateLimitWait is the maximum time to wait for a rate limit to reset
	MaxRateLimitWait = 5 * time.Minute
	// MaxRateLimitRetries is the maximum number of retries for rate limit errors after waiting
	MaxRateLimitRetries = 1
	// DefaultRateLimitWaitTime is the default time to wait for a rate limit to reset
	DefaultRateLimitWaitTime = 1 * time.Minute
)

// Github is the string that represents the GitHub provider
const Github = "github"

// Implements is the list of provider types that the GitHub provider implements
var Implements = []db.ProviderType{
	db.ProviderTypeGithub,
	db.ProviderTypeGit,
	db.ProviderTypeRest,
	db.ProviderTypeRepoLister,
}

// AuthorizationFlows is the list of authorization flows that the GitHub provider supports
var AuthorizationFlows = []db.AuthorizationFlow{
	db.AuthorizationFlowUserInput,
	db.AuthorizationFlowOauth2AuthorizationCodeFlow,
}

// GitHub is the struct that contains the GitHub client operations
type GitHub struct {
	client     *github.Client
	credential provifv1.GitHubCredential
	owner      string
	cache      ratecache.RestClientCache
}

// Ensure that the GitHub client implements the GitHub interface
var _ provifv1.GitHub = (*GitHub)(nil)

// NewRestClient creates a new GitHub REST API client
// BaseURL defaults to the public GitHub API, if needing to use a customer domain
// endpoint (as is the case with GitHub Enterprise), set the Endpoint field in
// the GitHubConfig struct
func NewRestClient(
	config *minderv1.GitHubProviderConfig,
	metrics telemetry.HttpClientMetrics,
	restClientCache ratecache.RestClientCache,
	credential provifv1.GitHubCredential,
	owner string,
) (*GitHub, error) {
	var err error

	tc := &http.Client{
		Transport: &oauth2.Transport{
			Base:   http.DefaultClient.Transport,
			Source: credential.GetAsOAuth2TokenSource(),
		},
	}

	tc.Transport, err = metrics.NewDurationRoundTripper(tc.Transport, db.ProviderTypeGithub)
	if err != nil {
		return nil, fmt.Errorf("error creating duration round tripper: %w", err)
	}

	ghClient := github.NewClient(tc)

	if config.Endpoint != "" {
		parsedURL, err := url.Parse(config.Endpoint)
		if err != nil {
			return nil, err
		}
		ghClient.BaseURL = parsedURL
	}

	return &GitHub{
		client:     ghClient,
		credential: credential,
		owner:      owner,
		cache:      restClientCache,
	}, nil
}

// ParseV1Config parses the raw config into a GitHubConfig struct
func ParseV1Config(rawCfg json.RawMessage) (*minderv1.GitHubProviderConfig, error) {
	type wrapper struct {
		GitHub *minderv1.GitHubProviderConfig `json:"github" yaml:"github" mapstructure:"github" validate:"required"`
	}

	var w wrapper
	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, err
	}

	// Validate the config according to the protobuf validation rules.
	if err := w.GitHub.Validate(); err != nil {
		return nil, fmt.Errorf("error validating GitHub v1 provider config: %w", err)
	}

	return w.GitHub, nil
}

// setAsRateLimited adds the GitHub to the cache as rate limited.
// An optimistic concurrency control mechanism is used to ensure that every request doesn't need
// synchronization. GitHub only adds itself to the cache if it's not already there. It doesn't
// remove itself from the cache when the rate limit is reset. This approach leverages the high
// likelihood of the client or token being rate-limited again. By keeping the client in the cache,
// we can reuse client's rateLimits map, which holds rate limits for different endpoints.
// This reuse of cached rate limits helps avoid unnecessary GitHub API requests when the client
// is rate-limited. Every cache entry has an expiration time, so the cache will eventually evict
// the rate-limited client.
func (c *GitHub) setAsRateLimited() {
	if c.cache != nil {
		c.cache.Set(c.owner, c.credential.GetCacheKey(), db.ProviderTypeGithub, c)
	}
}

// waitForRateLimitReset waits for token wait limit to reset. Returns error if wait time is more
// than MaxRateLimitWait or requests' context is cancelled.
func (c *GitHub) waitForRateLimitReset(ctx context.Context, err error) error {
	var rateLimitError *github.RateLimitError
	isRateLimitErr := errors.As(err, &rateLimitError)

	if isRateLimitErr {
		return c.processPrimaryRateLimitErr(ctx, rateLimitError)
	}

	var abuseRateLimitError *github.AbuseRateLimitError
	isAbuseRateLimitErr := errors.As(err, &abuseRateLimitError)

	if isAbuseRateLimitErr {
		return c.processAbuseRateLimitErr(ctx, abuseRateLimitError)
	}

	return nil
}

func (c *GitHub) processPrimaryRateLimitErr(ctx context.Context, err *github.RateLimitError) error {
	logger := zerolog.Ctx(ctx)
	rate := err.Rate
	if rate.Remaining == 0 {
		c.setAsRateLimited()

		waitTime := DefaultRateLimitWaitTime
		resetTime := rate.Reset.Time
		if !resetTime.IsZero() {
			waitTime = time.Until(resetTime)
		}

		logRateLimitError(logger, "RateLimitError", waitTime, c.owner, err.Response)

		if waitTime > MaxRateLimitWait {
			logger.Debug().Msgf("rate limit reset time: %v exceeds maximum wait time: %v", waitTime, MaxRateLimitWait)
			return err
		}

		// Wait for the rate limit to reset
		select {
		case <-time.After(waitTime):
			return nil
		case <-ctx.Done():
			logger.Debug().Err(ctx.Err()).Msg("context done while waiting for rate limit to reset")
			return err
		}
	}

	return nil
}

func (c *GitHub) processAbuseRateLimitErr(ctx context.Context, err *github.AbuseRateLimitError) error {
	logger := zerolog.Ctx(ctx)
	c.setAsRateLimited()

	retryAfter := err.RetryAfter
	waitTime := DefaultRateLimitWaitTime
	if retryAfter != nil && *retryAfter > 0 {
		waitTime = *retryAfter
	}

	logRateLimitError(logger, "AbuseRateLimitError", waitTime, c.owner, err.Response)

	if waitTime > MaxRateLimitWait {
		logger.Debug().Msgf("abuse rate limit wait time: %v exceeds maximum wait time: %v", waitTime, MaxRateLimitWait)
		return err
	}

	// Wait for the rate limit to reset
	select {
	case <-time.After(waitTime):
		return nil
	case <-ctx.Done():
		logger.Debug().Err(ctx.Err()).Msg("context done while waiting for rate limit to reset")
		return err
	}
}

func logRateLimitError(logger *zerolog.Logger, errType string, waitTime time.Duration, owner string, resp *http.Response) {
	var method, path string
	if resp != nil && resp.Request != nil {
		method = resp.Request.Method
		path = resp.Request.URL.Path
	}

	event := logger.Debug().
		Str("owner", owner).
		Str("wait_time", waitTime.String()).
		Str("error_type", errType)

	if method != "" {
		event = event.Str("method", method)
	}

	if path != "" {
		event = event.Str("path", path)
	}

	event.Msg("rate limit exceeded")
}

func performWithRetry[T any](ctx context.Context, op backoffv4.OperationWithData[T]) (T, error) {
	exponentialBackOff := backoffv4.NewExponentialBackOff()
	maxRetriesBackoff := backoffv4.WithMaxRetries(exponentialBackOff, MaxRateLimitRetries)
	return backoffv4.RetryWithData(op, backoffv4.WithContext(maxRetriesBackoff, ctx))
}

func isRateLimitError(err error) bool {
	var rateLimitError *github.RateLimitError
	isRateLimitErr := errors.As(err, &rateLimitError)

	var abuseRateLimitError *github.AbuseRateLimitError
	isAbuseRateLimitErr := errors.As(err, &abuseRateLimitError)

	return isRateLimitErr || isAbuseRateLimitErr
}
