// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"time"
)

// InvitationsConfig is the configuration for project invitations
type InvitationsConfig struct {
	// MaxExpiration is the maximum duration for which an invitation is valid.
	// Defaults to 168h (7 days).
	MaxExpiration time.Duration `mapstructure:"max_expiration" default:"168h"`
}
