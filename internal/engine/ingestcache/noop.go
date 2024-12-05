// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ingestcache

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// NoopCache is the interface for the ingest cache.
type NoopCache struct {
}

// NewNoopCache returns a new NoopCache
func NewNoopCache() Cache {
	return &NoopCache{}
}

// Get implements the Cache interface but does nothing
func (*NoopCache) Get(
	_ interfaces.Ingester,
	_ protoreflect.ProtoMessage,
	_ map[string]any,
) (*interfaces.Result, bool) {
	return nil, false
}

// Set implements the Cache interface but does nothing
func (*NoopCache) Set(
	_ interfaces.Ingester,
	_ protoreflect.ProtoMessage,
	_ map[string]any,
	_ *interfaces.Result,
) {
}
