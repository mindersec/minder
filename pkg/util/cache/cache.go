// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package cache contains cache utilities and implementations
package cache

// Cacher is the interface for the cache. It provides methods to get and set
type Cacher[T any] interface {
	Get(key string) (T, bool)
	Set(key string, value T)
}
