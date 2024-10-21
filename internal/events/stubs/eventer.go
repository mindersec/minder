// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package stubs contains stubs for the eventer package
package stubs

import (
	"context"
	"slices"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/mindersec/minder/internal/events"
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
