// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/rs/zerolog"
)

// pingEvent are messages sent from GitHub to check the status of a
// specific webhook. Minder's processing of these events consists in
// just reporting the source.
type pingEvent struct {
	HookID int64 `json:"hook_id,omitempty"`
	Repo   repo  `json:"repository,omitempty"`
	Sender user  `json:"sender,omitempty"`
}

// processPingEvent logs the type of token used to authenticate the
// webhook. The idea is to log a link between the repo and the token
// type. Since this is done only for the ping event, we can assume
// that the sender is the app that installed the webhook on the
// repository.
func processPingEvent(
	ctx context.Context,
	payload []byte,
) {
	l := zerolog.Ctx(ctx).With().Logger()

	var event pingEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		l.Info().Err(err).Msg("received malformed ping event")
		return
	}

	if event.Repo.ID != 0 {
		l = l.With().Int64("github-repository-id", event.Repo.ID).Logger()
		l = l.With().Str("github-repository-url", event.Repo.HTMLURL).Logger()
	}
	if event.Sender.Login != "" {
		l = l.With().Str("sender-login", event.Sender.Login).Logger()
		l = l.With().Str("github-repository-url", event.Sender.HTMLURL).Logger()
		if strings.Contains(event.Sender.HTMLURL, "github.com/apps") {
			l = l.With().Str("sender-token-type", "github-app").Logger()
		} else {
			l = l.With().Str("sender-token-type", "oauth-app").Logger()
		}
	}

	l.Debug().Msg("ping received")
}
