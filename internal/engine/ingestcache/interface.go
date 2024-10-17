// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ingestcache a cache that is used to cache the results of ingesting data.
// The intent is to reduce the number of calls to external services.
package ingestcache

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// Cache is the interface for the ingest cache.
type Cache interface {
	Get(ingester interfaces.Ingester, entity protoreflect.ProtoMessage, params map[string]any) (*interfaces.Result, bool)
	Set(ingester interfaces.Ingester, entity protoreflect.ProtoMessage, params map[string]any, result *interfaces.Result)
}
