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

// Package manager provides the manager for the provider classes
package manager

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=../../../../internal/providers/manager/mock/if_$GOFILE -source=./$GOFILE

// ProviderClassManager describes an interface for creating instances of a
// specific Provider class. The idea is that ProviderManager determines the
// class of the Provider, and delegates to the appropraite ProviderClassManager
type ProviderClassManager interface {
	// MarshallConfig validates the config and marshalls it into a format that can be stored in the database
	MarshallConfig(ctx context.Context, class db.ProviderClass, config json.RawMessage) (json.RawMessage, error)
	// Build creates an instance of Provider based on the config in the DB
	Build(ctx context.Context, config *db.Provider) (v1.Provider, error)
	// Delete deletes an instance of this provider
	Delete(ctx context.Context, config *db.Provider) error
	// GetSupportedClasses lists the types of Provider class which this manager
	// can produce.
	GetSupportedClasses() []db.ProviderClass
	// GetWebhookHandler returns the webhook handler for the provider class
	GetWebhookHandler() http.Handler
}
