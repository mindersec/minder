// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package eventer provides the constructor for the eventer
package eventer

import (
	"context"

	"github.com/mindersec/minder/internal/events"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// New creates a new eventer object implementing the Interface interface
func New(ctx context.Context, cfg *serverconfig.EventConfig) (interfaces.Interface, error) {
	return events.NewEventer(ctx, cfg)
}
