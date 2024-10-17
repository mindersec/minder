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
	HookID *int64 `json:"hook_id,omitempty"`
	Repo   *repo  `json:"repository,omitempty"`
	Sender *user  `json:"sender,omitempty"`
}

func (p *pingEvent) GetRepo() *repo {
	return p.Repo
}

func (p *pingEvent) GetHookID() int64 {
	if p.HookID != nil {
		return *p.HookID
	}
	return 0
}

func (p *pingEvent) GetSender() *user {
	return p.Sender
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

	var event *pingEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		l.Info().Err(err).Msg("received malformed ping event")
		return
	}

	if event.GetRepo() != nil {
		l = l.With().Int64("github-repository-id", event.GetRepo().GetID()).Logger()
		l = l.With().Str("github-repository-url", event.GetRepo().GetHTMLURL()).Logger()
	}
	if event.GetSender() != nil {
		l = l.With().Str("sender-login", event.GetSender().GetLogin()).Logger()
		l = l.With().Str("github-repository-url", event.GetSender().GetHTMLURL()).Logger()
		if strings.Contains(event.GetSender().GetHTMLURL(), "github.com/apps") {
			l = l.With().Str("sender-token-type", "github-app").Logger()
		} else {
			l = l.With().Str("sender-token-type", "oauth-app").Logger()
		}
	}

	l.Debug().Msg("ping received")
}
