//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package unordered providers a handler wrapper that will handle events in an unordered fashion.
// This means that events will be acked immediately, and the handler will not wait for the event to be processed.
// If a retry is needed, the event will be requeued and processed again.
package unordered

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/events"
)

func TestUnorderedProcessor_Wrap_HandlesEvent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	evt, err := events.Setup(ctx, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err)

	u := New(evt)

	var counter atomic.Uint32

	h := u.Wrap("test", func(_ *message.Message) error {
		counter.Add(1)
		return nil
	})

	done := make(chan struct{})

	evt.Register("test", h, func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			msgs, err := h(msg)
			done <- struct{}{}
			return msgs, err
		}
	})

	go func() {
		err := evt.Run(ctx)
		assert.NoError(t, err)
	}()

	<-evt.Running()

	assert.NoError(t, evt.Publish("test", message.NewMessage(uuid.New().String(), nil)))

	<-done

	require.NoError(t, evt.Close(), "closing eventer")

	u.Wait()

	assert.Equal(t, uint32(1), counter.Load())
}

func TestUnorderedProcessor_Wrap_HandlesErrors(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	evt, err := events.Setup(ctx, &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err)

	u := New(evt)

	var counter atomic.Uint32

	h := u.Wrap("test", func(msg *message.Message) error {
		counter.Add(1)
		t.Logf("handling message %s", msg.UUID)
		t.Logf("metadata: %v", msg.Metadata)

		return errors.New("some error")
	})

	done := make(chan struct{}, 3)

	evt.Register("test", h, func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			msgs, err := h(msg)
			done <- struct{}{}
			return msgs, err
		}
	})

	go func() {
		err := evt.Run(ctx)
		assert.NoError(t, err)
	}()

	<-evt.Running()

	assert.NoError(t, evt.Publish("test", message.NewMessage(uuid.New().String(), nil)))

	<-done

	time.Sleep(2 * time.Second)

	u.Wait()

	require.NoError(t, evt.Close(), "closing eventer")

	assert.Equal(t, uint32(4), counter.Load(), "expected 3 retries (A counter of 4)")
}
