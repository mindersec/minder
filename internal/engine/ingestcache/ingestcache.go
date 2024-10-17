// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ingestcache

import (
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	engif "github.com/mindersec/minder/internal/engine/interfaces"
)

// ErrBuildingCacheKey is the error returned when building a cache key fails
var ErrBuildingCacheKey = errors.New("error building cache key")

type cache struct {
	// cache is the actual cache
	cache *xsync.MapOf[string, *engif.Result]
}

// NewCache returns a new cache
func NewCache() Cache {
	return &cache{
		cache: xsync.NewMapOf[string, *engif.Result](),
	}
}

// Get attempts to get a result from the cache
func (c *cache) Get(
	ingester engif.Ingester,
	entity protoreflect.ProtoMessage,
	params map[string]any,
) (*engif.Result, bool) {
	key, err := buildCacheKey(ingester, entity, params)
	if err != nil {
		// TODO we might want to log this
		log.Printf("error building cache key: %v", err)
		return nil, false
	}

	return c.cache.Load(key)
}

// Set sets a result in the cache
func (c *cache) Set(
	ingester engif.Ingester,
	entity protoreflect.ProtoMessage,
	params map[string]any,
	result *engif.Result,
) {
	key, err := buildCacheKey(ingester, entity, params)
	if err != nil {
		// TODO we might want to log this
		log.Printf("error building cache key: %v", err)
		return
	}

	c.cache.Store(key, result)
}

func buildCacheKey(
	ingester engif.Ingester,
	entity protoreflect.ProtoMessage,
	ruleparams map[string]any,
) (string, error) {
	chsum := sha512.New()

	_, err := chsum.Write([]byte(ingester.GetType()))
	if err != nil {
		return "", fmt.Errorf("%w: couldn't checksum type: %v", ErrBuildingCacheKey, err)
	}

	if ingester.GetConfig() != nil {
		marshaledcfg, err := protojson.Marshal(ingester.GetConfig())
		if err != nil {
			return "", fmt.Errorf("%w: couldn't marshal config: %v", ErrBuildingCacheKey, err)
		}

		if _, err := chsum.Write(marshaledcfg); err != nil {
			return "", fmt.Errorf("%w: couldn't checksum config: %v", ErrBuildingCacheKey, err)
		}
	}

	marshaleldEntity, err := protojson.Marshal(entity)
	if err != nil {
		return "", fmt.Errorf("%w: couldn't marshal entity: %v", ErrBuildingCacheKey, err)
	}

	if _, err := chsum.Write(marshaleldEntity); err != nil {
		return "", fmt.Errorf("%w: couldn't checksum entity: %v", ErrBuildingCacheKey, err)
	}

	if ruleparams != nil {
		marshaleldParams, err := json.Marshal(ruleparams)
		if err != nil {
			return "", fmt.Errorf("%w: couldn't marshal rule params: %v", ErrBuildingCacheKey, err)
		}

		if _, err := chsum.Write(marshaleldParams); err != nil {
			return "", fmt.Errorf("%w: couldn't checksum rule params: %v", ErrBuildingCacheKey, err)
		}
	}

	return string(chsum.Sum(nil)), nil
}
