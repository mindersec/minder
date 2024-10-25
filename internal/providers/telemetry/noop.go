// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"net/http"

	"github.com/mindersec/minder/pkg/db"
)

type noop struct{}

var _ ProviderMetrics = (*noop)(nil)

// NewNoopMetrics returns a new noop httpClientMetrics provider
func NewNoopMetrics() *noop {
	return &noop{}
}

func (_ *noop) NewDurationRoundTripper(wrapped http.RoundTripper, _ db.ProviderType) (http.RoundTripper, error) {
	return wrapped, nil
}
