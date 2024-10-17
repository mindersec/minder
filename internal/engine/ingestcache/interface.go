// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ingestcache a cache that is used to cache the results of ingesting data.
// The intent is to reduce the number of calls to external services.
package ingestcache

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	engif "github.com/mindersec/minder/internal/engine/interfaces"
)

// Cache is the interface for the ingest cache.
type Cache interface {
	Get(ingester engif.Ingester, entity protoreflect.ProtoMessage, params map[string]any) (*engif.Result, bool)
	Set(ingester engif.Ingester, entity protoreflect.ProtoMessage, params map[string]any, result *engif.Result)
}
