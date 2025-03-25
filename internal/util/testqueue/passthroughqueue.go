// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package testqueue contains queue utilities for testing
package testqueue

import (
	"sync"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
)

// PassthroughQueue is a queue that passes messages through.
// It's only useful for testing.
type PassthroughQueue struct {
	ch     chan *message.Message
	chLock sync.Mutex
	t      *testing.T
}

// NewPassthroughQueue creates a new PassthroughQueue
func NewPassthroughQueue(t *testing.T) *PassthroughQueue {
	t.Helper()

	return &PassthroughQueue{
		ch: make(chan *message.Message),
		t:  t,
	}
}

// GetQueue returns the queue
func (q *PassthroughQueue) GetQueue() <-chan *message.Message {
	q.chLock.Lock()
	defer q.chLock.Unlock()
	return q.ch
}

// Pass passes a message through the queue
func (q *PassthroughQueue) Pass(msg *message.Message) error {
	q.t.Logf("Passing message through queue: %s", msg.UUID)
	q.chLock.Lock()
	defer q.chLock.Unlock()
	q.ch <- msg
	return nil
}

// Close frees closes the channel used as queue.
func (q *PassthroughQueue) Close() error {
	q.chLock.Lock()
	defer q.chLock.Unlock()
	close(q.ch)
	return nil
}
