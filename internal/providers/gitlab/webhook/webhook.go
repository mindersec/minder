//
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

// Package webhook provides the GitLab webhook implementation
package webhook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
)

type handler struct {
	l zerolog.Logger
}

// NewHandler creates a new webhook handler
func NewHandler(l zerolog.Logger) *handler {
	return &handler{
		l: decorateLogger(l),
	}
}

// WebHook processes the incoming GitLab webhook event
func (h *handler) WebHook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var event map[string]interface{}
		if err := json.Unmarshal(body, &event); err != nil {
			http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
			return
		}

		prettyEvent, err := json.MarshalIndent(event, "", "  ")
		if err != nil {
			http.Error(w, "Failed to pretty print event", http.StatusInternalServerError)
			return
		}

		h.l.Debug().Str("event", string(prettyEvent)).Msg("Received webhook event")

		// Process the event based on its type
		eventType := r.Header.Get("X-Gitlab-Event")
		switch eventType {
		case "Push Hook":
			h.l.Debug().Str("event_type", eventType).Msg("Received push webhook event")
		case "Merge Request Hook":
			h.l.Debug().Str("event_type", eventType).Msg("Received merge webhook event")
		// Add more event types as needed
		default:
			h.l.Debug().Str("event_type", eventType).Msg("Received unknown webhook event")
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Webhook event received")
	}
}
