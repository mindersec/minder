// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	gitlablib "github.com/xanzy/go-gitlab"

	"github.com/mindersec/minder/internal/providers/gitlab/webhooksecret"
)

const (
	// MaxBytesLimit is the maximum number of bytes to read from the response body
	// We limit to 1MB to prevent abuse
	MaxBytesLimit int64 = 1 << 20
)

// GetWebhookHandler implements the ProviderManager interface
// Note that this is where the whole webhook handler is defined and
// will live.
func (m *providerClassManager) GetWebhookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := zerolog.Ctx(m.parentContext).With().
			Str("webhook", "gitlab").
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote", r.RemoteAddr).
			Str("user-agent", r.UserAgent()).
			Str("content-type", r.Header.Get("Content-Type")).
			Logger()

		// Validate the webhook secret
		if err := m.validateRequest(r); err != nil {
			l.Error().Err(err).Msg("invalid webhook request")
			http.Error(w, "invalid webhook request", http.StatusUnauthorized)
			return
		}

		eventType := gitlablib.HookEventType(r)
		if eventType == "" {
			l.Error().Msg("missing X-Gitlab-Event header")
			http.Error(w, "missing X-Gitlab-Event header", http.StatusBadRequest)
			return
		}

		l = l.With().Str("event", string(eventType)).Logger()

		disp := m.getWebhookEventDispatcher(eventType)

		if err := disp(l, r); err != nil {
			l.Error().Err(err).Msg("error handling webhook event")
			http.Error(w, "error handling webhook event", http.StatusInternalServerError)
			return
		}

		l.Debug().Msg("processed webhook event successfully")
	})
}

// getWebhookEventDispatcher returns the appropriate webhook event dispatcher for the given event type
// It returns a function that is meant to do the actual handling of the event.
// Note that we pass the request to the handler function, so we don't even try to
// parse the request body here unless it's necessary.
func (m *providerClassManager) getWebhookEventDispatcher(
	eventType gitlablib.EventType,
) func(l zerolog.Logger, r *http.Request) error {
	//nolint:exhaustive // We only handle a subset of the possible events
	switch eventType {
	case gitlablib.EventTypePush:
		return m.handleRepoPush
	case gitlablib.EventTypeTagPush:
		return m.handleTagPush
	case gitlablib.EventTypeMergeRequest:
		return m.handleMergeRequest
	case gitlablib.EventTypeRelease:
		return m.handleRelease
	default:
		return m.handleNoop
	}
}

// handleNoop is a no-op handler for unhandled webhook events
func (_ *providerClassManager) handleNoop(l zerolog.Logger, _ *http.Request) error {
	l.Debug().Msg("unhandled webhook event")
	return nil
}

func (m *providerClassManager) validateRequest(r *http.Request) error {
	// Validate the webhook secret
	gltok := gitlablib.HookEventToken(r)
	if gltok == "" {
		return errors.New("missing X-Gitlab-Token header")
	}

	if err := m.validateToken(gltok, r); err != nil {
		return fmt.Errorf("invalid X-Gitlab-Token header: %w", err)
	}

	return nil
}

// validateToken validates the incoming GitLab webhook token
// Validation takes the secret from the GitLab webhook configuration
// appens the last element of the path to the URL (which is unique per entity)
func (m *providerClassManager) validateToken(token string, req *http.Request) error {
	// Extract the unique ID from the URL path
	path := req.URL.Path
	uniq := path[strings.LastIndex(path, "/")+1:]

	// uniq must be a valid UUID
	_, err := uuid.Parse(uniq)
	if err != nil {
		return errors.New("invalid unique ID")
	}

	// Generate the expected secret
	if valid := webhooksecret.Verify(m.currentWebhookSecret, uniq, token); valid {
		// If the secret is valid, we can return
		return nil
	}

	// Check the previous secrets
	for _, prev := range m.previousWebhookSecrets {
		if valid := webhooksecret.Verify(prev, uniq, token); valid {
			return nil
		}
	}

	return errors.New("invalid webhook token")
}

func decodeJSONSafe[T any](r io.ReadCloser, v *T) error {
	rs := wrapSafe(r)
	defer r.Close()

	dec := json.NewDecoder(rs)
	return dec.Decode(v)
}

// wrapSafe wraps the io.Reader in a LimitReader to prevent abuse
func wrapSafe(r io.Reader) io.Reader {
	return io.LimitReader(r, MaxBytesLimit)
}
