// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package manager contains the GitHubProviderClassManager
package manager

import (
	"net/http"

	"github.com/mindersec/minder/internal/providers/github/webhook"
)

// GetWebhookHandler implements the ProviderManager interface
// Note that this is where the whole webhook handler is defined and
// will live.
func (mgr *githubProviderManager) GetWebhookHandler() http.Handler {
	return webhook.HandleWebhookEvent(mgr.mt, mgr.publisher, mgr.whconfig)
}
