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

// Package gochannel provides a gochannel implementation of the eventer
package gochannel

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/events/common"
)

// BuildGoChannelDriver creates a gochannel driver for the eventer
func BuildGoChannelDriver(cfg *serverconfig.EventConfig) (message.Publisher, message.Subscriber, common.DriverCloser, error) {
	pubsub := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer: cfg.GoChannel.BufferSize,
		Persistent:          cfg.GoChannel.PersistEvents,
	}, nil)

	return pubsub, pubsub, func() {}, nil
}
