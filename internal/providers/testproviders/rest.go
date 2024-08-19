// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testproviders

import (
	"context"

	"github.com/stacklok/minder/internal/entities/properties"
	"github.com/stacklok/minder/internal/providers/http"
	"github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// RESTProvider is a test implementation of the REST provider
// interface
type RESTProvider struct {
	*http.REST
}

// NewRESTProvider creates a new REST provider
func NewRESTProvider(
	config *minderv1.RESTProviderConfig,
	metrics telemetry.HttpClientMetrics,
	credential provifv1.RestCredential,
) (*RESTProvider, error) {
	r, err := http.NewREST(config, metrics, credential)
	if err != nil {
		return nil, err
	}
	return &RESTProvider{
		REST: r,
	}, nil
}

// Ensure RESTProvider implements the Provider interface
var _ provifv1.Provider = (*RESTProvider)(nil)

// CanImplement implements the Provider interface
func (_ *RESTProvider) CanImplement(trait minderv1.ProviderType) bool {
	return trait == minderv1.ProviderType_PROVIDER_TYPE_REST
}

// FetchAllProperties implements the Provider interface
func (_ *RESTProvider) FetchAllProperties(
	_ context.Context, _ string, _ minderv1.Entity) (*properties.Properties, error) {
	return nil, nil
}

// FetchProperty implements the Provider interface
func (_ *RESTProvider) FetchProperty(
	_ context.Context, _ string, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}
