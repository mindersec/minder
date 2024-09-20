// Copyright 2024 Stacklok, Inc.
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

package manager

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	gitlablib "github.com/xanzy/go-gitlab"

	"github.com/stacklok/minder/internal/providers/gitlab/webhooksecret"
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
