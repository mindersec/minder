// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"context"
	"errors"
	"sync"

	engif "github.com/mindersec/minder/internal/engine/interfaces"
)

// SharedActionsContextKey is the key used to store the shared actions context
// in the context.Context.
type SharedActionsContextKey struct{}

// SharedFlusherKey is the key used to store the shared flusher
type SharedFlusherKey string

type sharedFlusher struct {
	flusher engif.AggregatingAction
	items   []any
}

// SharedActionsContext is the shared actions context.
type SharedActionsContext struct {
	shared map[SharedFlusherKey]*sharedFlusher
	mux    sync.Mutex
}

// WithSharedActionsContext returns a new context.Context with the shared actions
// context set.
func WithSharedActionsContext(ctx context.Context) (context.Context, *SharedActionsContext) {
	sac := &SharedActionsContext{
		shared: make(map[SharedFlusherKey]*sharedFlusher),
	}
	return context.WithValue(ctx, SharedActionsContextKey{}, sac), sac
}

// GetSharedActionsContext returns the shared actions context from the context.Context.
func GetSharedActionsContext(ctx context.Context) *SharedActionsContext {
	ctxVal := ctx.Value(SharedActionsContextKey{})
	if ctxVal == nil {
		return nil
	}

	v, ok := ctxVal.(*SharedActionsContext)
	if !ok {
		return nil
	}

	return v
}

// ShareAndRegister adds a shared value to the shared actions context. It may
// also register a flusher if it does not exist.
func (sac *SharedActionsContext) ShareAndRegister(key SharedFlusherKey, flusher engif.AggregatingAction, item any) {
	sac.mux.Lock()
	defer sac.mux.Unlock()

	f, ok := sac.shared[key]
	if !ok {
		f = &sharedFlusher{
			flusher: flusher,
			items:   []any{item},
		}
		sac.shared[key] = f
		return
	}

	f.items = append(f.items, item)
}

// Flush returns all the shared values and clears the shared actions context.
func (sac *SharedActionsContext) Flush(ctx context.Context) error {
	sac.mux.Lock()
	defer sac.mux.Unlock()
	var errs []error

	for key, f := range sac.shared {
		err := f.flusher.Flush(ctx, f.items...)
		if err != nil {
			errs = append(errs, err)
		}

		delete(sac.shared, key)
	}

	return errors.Join(errs...)
}
