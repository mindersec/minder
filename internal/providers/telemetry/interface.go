// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package telemetry provides the telemetry interfaces and implementations for providers
package telemetry

import (
	"net/http"

	"github.com/mindersec/minder/internal/db"
)

// HttpClientMetrics provides the httpClientMetrics for http clients
type HttpClientMetrics interface {
	NewDurationRoundTripper(wrapped http.RoundTripper, providerType db.ProviderType) (http.RoundTripper, error)
}

// ProviderMetrics provides the httpClientMetrics for providers
type ProviderMetrics interface {
	HttpClientMetrics
}
