//
// Copyright 2023 Stacklok, Inc.
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

// Package controlplane contains the control plane API for the minder.
package controlplane

import (
	"errors"
	"github.com/stacklok/minder/internal/webhooks/handlers"
	"net/http"
)

// HandleWebhook receives the request and passes it to the dispatcher for processing
func (s *Server) HandleWebhook() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		err := s.webhooks.Dispatch(request.Context(), request)
		if errors.Is(err, handlers.ErrCantParse) {
			writer.WriteHeader(http.StatusOK)
		} else if err != nil {
			// TODO: more fine grained error handling
			writer.WriteHeader(http.StatusInternalServerError)
		} else {
			writer.WriteHeader(http.StatusOK)
		}
		return
	}
}
