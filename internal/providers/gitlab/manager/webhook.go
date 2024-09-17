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
	"net/http"

	"github.com/rs/zerolog"
)

// GetWebhookHandler implements the ProviderManager interface
// Note that this is where the whole webhook handler is defined and
// will live.
func (m *providerClassManager) GetWebhookHandler() http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		l := zerolog.Ctx(m.parentContext).With().
			Str("webhook", "gitlab").
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote", r.RemoteAddr).
			Str("user-agent", r.UserAgent()).
			Str("content-type", r.Header.Get("Content-Type")).
			Logger()

		// TODO: Implement webhook handler

		l.Debug().Msg("received webhook")
	})
}
