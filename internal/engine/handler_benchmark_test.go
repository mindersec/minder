// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/mindersec/minder/internal/engine/entities"
)

// fakeExecutor is a minimal stub implementation of the Executor interface.
// It allows the benchmark to run without invoking real evaluation logic.
type fakeExecutor struct{}

// EvalEntityEvent satisfies the Executor interface.
// It performs no work and always returns nil.
func (f *fakeExecutor) EvalEntityEvent(_ context.Context, _ *entities.EntityInfoWrapper) error {
	_ = f
	return nil
}

// newTestHandler creates a minimal ExecutorEventHandler instance
// configured with a fake executor for benchmarking purposes.
func newTestHandler() *ExecutorEventHandler {
	return &ExecutorEventHandler{
		executor: &fakeExecutor{},
	}
}

// newTestMessage creates a minimal Watermill message with a background context.
func newTestMessage() *message.Message {
	msg := message.NewMessage("test-id", []byte("{}"))
	msg.SetContext(context.Background())
	return msg
}

// BenchmarkHandleEntityEvent measures the sequential performance of
// ExecutorEventHandler.HandleEntityEvent.
//
// This benchmark provides a baseline for:
// - execution time per event
// - memory allocations per operation
//
// It isolates handler logic by using a fake executor.
func BenchmarkHandleEntityEvent(b *testing.B) {
	handler := newTestHandler()
	msg := newTestMessage()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler.HandleEntityEvent(msg)
	}
}

// BenchmarkHandleEntityEventParallel measures performance under parallel load.
//
// This helps evaluate how the handler behaves when multiple events
// are processed concurrently, which is important for throughput analysis.
func BenchmarkHandleEntityEventParallel(b *testing.B) {
	handler := newTestHandler()
	msg := newTestMessage()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = handler.HandleEntityEvent(msg)
		}
	})
}
