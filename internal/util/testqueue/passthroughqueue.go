//
// Copyright 2023 Stacklok, Inc.
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

// Package testqueue contains queue utilities for testing
package testqueue

import "github.com/ThreeDotsLabs/watermill/message"

// PassthroughQueue is a queue that passes messages through.
// It's only useful for testing.
type PassthroughQueue struct {
	ch chan *message.Message
}

// NewPassthroughQueue creates a new PassthroughQueue
func NewPassthroughQueue() *PassthroughQueue {
	return &PassthroughQueue{
		ch: make(chan *message.Message),
	}
}

// GetQueue returns the queue
func (q *PassthroughQueue) GetQueue() <-chan *message.Message {
	return q.ch
}

// Pass passes a message through the queue
func (q *PassthroughQueue) Pass(msg *message.Message) error {
	q.ch <- msg
	return nil
}
