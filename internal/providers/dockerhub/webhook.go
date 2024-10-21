// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

import "net/http"

// GetWebhookHandler implements the ProviderManager interface
// Note that this is where the whole webhook handler is defined and
// will live.
func (_ *providerClassManager) GetWebhookHandler() http.Handler {
	return nil
}
