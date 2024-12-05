// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package eventer provides an interface for creating a new eventer
package eventer

import (
	"context"

	"github.com/open-feature/go-sdk/openfeature"

	"github.com/mindersec/minder/internal/events"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// New creates a new eventer
func New(ctx context.Context, flagClient openfeature.IClient, cfg *serverconfig.EventConfig) (interfaces.Interface, error) {
	return events.NewEventer(ctx, flagClient, cfg)
}
