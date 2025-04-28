// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import "github.com/mindersec/minder/pkg/engine/v1/interfaces"

// VoidResultSink is a result sink that does nothing
type VoidResultSink struct{}

// NewVoidResultSink creates a new void result sink
func NewVoidResultSink() *VoidResultSink {
	return &VoidResultSink{}
}

// SetIngestResult implements the ResultSink interface
func (VoidResultSink) SetIngestResult(_ *interfaces.Ingested) {
}
