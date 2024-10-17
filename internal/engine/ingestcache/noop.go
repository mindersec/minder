// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ingestcache

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	engif "github.com/mindersec/minder/internal/engine/interfaces"
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
	_ engif.Ingester,
	_ protoreflect.ProtoMessage,
	_ map[string]any,
) (*engif.Result, bool) {
	return nil, false
}

// Set implements the Cache interface but does nothing
func (*NoopCache) Set(
	_ engif.Ingester,
	_ protoreflect.ProtoMessage,
	_ map[string]any,
	_ *engif.Result,
) {
}
