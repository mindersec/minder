// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
)

func TestNewNoopMetrics(t *testing.T) {
	t.Parallel()
	m := NewNoopMetrics()
	assert.NotNil(t, m)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mockdb.NewMockStore(ctrl)

	// No-op methods should not panic
	err := m.Init(mockStore)
	assert.NoError(t, err)

	m.AddWebhookEventTypeCount(context.Background(), &WebhookEventState{Typ: "test"})
	m.AddTokenOpCount(context.Background(), "test", true)
}

func TestNewMetrics(t *testing.T) {
	t.Parallel()
	m := NewMetrics()
	assert.NotNil(t, m)
}
