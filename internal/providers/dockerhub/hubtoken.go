// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// hubAuthTokenURL exchanges a Docker Hub personal access token for the JWT
// that the Docker Hub API (hub.docker.com/v2) expects as a Bearer token.
const hubAuthTokenURL = "https://hub.docker.com/v2/auth/token" //nolint:gosec // G101: an endpoint URL, not a credential

const (
	// hubTokenFallbackTTL is used when the returned JWT has no readable
	// iat/exp claims.
	hubTokenFallbackTTL = 5 * time.Minute
	// hubTokenMaxTTL caps how long a JWT is reused, so a token Docker Hub
	// revokes early is replaced within this time.
	hubTokenMaxTTL = 15 * time.Minute
	// maxTokenResponseSize bounds how much of the token response is read.
	maxTokenResponseSize = 64 << 10
)

// hubTokenSource is an oauth2.TokenSource that exchanges the provider's
// personal access token for a Docker Hub API JWT. The Docker Hub API rejects
// a personal access token sent directly as a Bearer token.
type hubTokenSource struct {
	cli      *http.Client
	tokenURL string
	username string
	pat      oauth2.TokenSource
}

// newHubClient returns an HTTP client for the Docker Hub API that
// authenticates with a JWT obtained from tokenURL, reusing it until it expires.
func newHubClient(pat oauth2.TokenSource, username, tokenURL string) *http.Client {
	src := &hubTokenSource{
		cli:      &http.Client{Timeout: 30 * time.Second},
		tokenURL: tokenURL,
		username: username,
		pat:      pat,
	}
	return oauth2.NewClient(context.Background(), oauth2.ReuseTokenSource(nil, src))
}

// Token implements oauth2.TokenSource.
func (s *hubTokenSource) Token() (*oauth2.Token, error) {
	pat, err := s.pat.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get personal access token: %w", err)
	}

	// The token endpoint takes the personal access token in the request body.
	body, err := json.Marshal(struct { //nolint:gosec // G117: sending the secret here is the point of the exchange
		Identifier string `json:"identifier"`
		Secret     string `json:"secret"`
	}{Identifier: s.username, Secret: pat.AccessToken})
	if err != nil {
		return nil, fmt.Errorf("error encoding token request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.tokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error requesting Docker Hub API token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker hub token exchange failed: %s", resp.Status)
	}

	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxTokenResponseSize)).Decode(&out); err != nil {
		return nil, fmt.Errorf("error decoding Docker Hub token response: %w", err)
	}
	if out.AccessToken == "" {
		return nil, fmt.Errorf("docker hub token response has no access_token")
	}

	return &oauth2.Token{
		AccessToken: out.AccessToken,
		TokenType:   "Bearer",
		Expiry:      jwtExpiry(out.AccessToken, time.Now()),
	}, nil
}

// jwtExpiry returns when to stop reusing a JWT. It reads the iat and exp
// claims without verifying them and applies their difference, the lifetime
// Docker Hub gave the token, from the local now. That keeps clock skew between
// this host and Docker Hub out of the result. The lifetime is capped at
// hubTokenMaxTTL, and hubTokenFallbackTTL is used when the claims can't be read.
func jwtExpiry(token string, now time.Time) time.Time {
	ttl := hubTokenFallbackTTL
	if parts := strings.Split(token, "."); len(parts) == 3 {
		if payload, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			var claims struct {
				Iat int64 `json:"iat"`
				Exp int64 `json:"exp"`
			}
			if json.Unmarshal(payload, &claims) == nil && claims.Iat > 0 && claims.Exp > claims.Iat {
				ttl = min(time.Duration(claims.Exp-claims.Iat)*time.Second, hubTokenMaxTTL)
			}
		}
	}
	return now.Add(ttl)
}
