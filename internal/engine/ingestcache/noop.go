// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ingestcache

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	engif "github.com/stacklok/minder/internal/engine/interfaces"
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
