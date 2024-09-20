// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package stubs contains stubs for the eventer package
package stubs

import (
	"context"
	"slices"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/stacklok/minder/internal/events"
)

// StubEventer is a stub implementation of events.Interface and the events.Publisher interface
var _ events.Interface = (*StubEventer)(nil)
var _ events.Publisher = (*StubEventer)(nil)

// StubEventer is an eventer that's useful for testing.
type StubEventer struct {
	Topics []string
	Sent   []*message.Message
}

// Close implements events.Interface.
func (*StubEventer) Close() error {
	panic("unimplemented")
}

// ConsumeEvents implements events.Interface.
func (*StubEventer) ConsumeEvents(...events.Consumer) {
	panic("unimplemented")
}

// Publish implements events.Interface.
func (s *StubEventer) Publish(topic string, messages ...*message.Message) error {
	if !slices.Contains(s.Topics, topic) {
		s.Topics = append(s.Topics, topic)
	}
	s.Sent = append(s.Sent, messages...)
	return nil
}

// Register implements events.Interface.
func (*StubEventer) Register(string, message.NoPublishHandlerFunc, ...message.HandlerMiddleware) {
	panic("unimplemented")
}

// Run implements events.Interface.
func (*StubEventer) Run(context.Context) error {
	panic("unimplemented")
}

// Running implements events.Interface.
func (*StubEventer) Running() chan struct{} {
	panic("unimplemented")
}
