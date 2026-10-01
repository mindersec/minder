// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package meters

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"
)

func TestExportingMeterFactory(t *testing.T) {
	t.Parallel()

	factory := &ExportingMeterFactory{}
	meter := factory.Build("test-meter")

	require.NotNil(t, meter, "Expected meter to not be nil")
	_, isNoop := meter.(noop.Meter)
	require.False(t, isNoop, "Expected exporting meter not to be a noop.Meter")
}

func TestNoopMeterFactory(t *testing.T) {
	t.Parallel()

	factory := &NoopMeterFactory{}
	meter := factory.Build("test-meter")

	require.NotNil(t, meter, "Expected noop meter to not be nil")
	_, isNoop := meter.(noop.Meter)
	require.True(t, isNoop, "Expected meter to be of type noop.Meter")
}
