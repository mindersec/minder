// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package quay

import "net/http"

// GetWebhookHandler implements the ProviderManager interface
// Note that this is where the whole webhook handler is defined and
// will live.
func (*providerClassManager) GetWebhookHandler() http.Handler {
	return nil
}
