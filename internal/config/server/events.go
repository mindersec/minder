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

package server

import (
	"time"

	"github.com/stacklok/minder/internal/config"
)

// EventConfig is the configuration for minder's eventing system.
type EventConfig struct {
	// Driver is the driver used to store events
	Driver string `mapstructure:"driver" default:"go-channel"`
	// RouterCloseTimeout is the timeout for closing the router in seconds
	RouterCloseTimeout int64 `mapstructure:"router_close_timeout" default:"10"`
	// GoChannel is the configuration for the go channel event driver
	GoChannel GoChannelEventConfig `mapstructure:"go-channel"`
	// SQLPubSub is the configuration for the database event driver
	SQLPubSub SQLEventConfig `mapstructure:"sql"`
	// Aggregator is the configuration for the event aggregator middleware
	Aggregator AggregatorConfig `mapstructure:"aggregator"`
	// Nats is the configuration when using NATS as the event driver
	Nats NatsConfig `mapstructure:"nats"`
}

// GoChannelEventConfig is the configuration for the go channel event driver
// for minder's eventing system.
type GoChannelEventConfig struct {
	// BufferSize is the size of the buffer for the go channel
	BufferSize int64 `mapstructure:"buffer_size" default:"0"`
	// PersistEvents is whether or not to persist events to the channel
	PersistEvents bool `mapstructure:"persist_events" default:"false"`
	// BlockPublishUntilSubscriberAck is whether or not to block publishing until
	// the subscriber acks the message. This is useful for testing.
	BlockPublishUntilSubscriberAck bool `mapstructure:"block_publish_until_subscriber_ack" default:"false"`
}

// SQLEventConfig is the configuration for the database event driver
type SQLEventConfig struct {
	// InitSchema is whether or not to initialize the schema
	InitSchema bool                  `mapstructure:"init_schema" default:"true"`
	Connection config.DatabaseConfig `mapstructure:"connection" default:"{\"dbname\":\"watermill\"}"`
	// AckDeadline is the deadline (in seconds) before timing out and re-attempting
	// a message delivery.  Note that setting this too short can cause messages to
	// be retried even they should be marked as "poison".
	AckDeadline time.Duration `mapstructure:"ack_deadline" default:"300s"`
}

// AggregatorConfig is the configuration for the event aggregator middleware
type AggregatorConfig struct {
	// LockInterval is the interval for locking events in seconds.
	// This is the threshold between rule evaluations + actions.
	LockInterval int64 `mapstructure:"lock_interval" default:"30"`
}

// NatsConfig is the configuration when using NATS as the event driver
type NatsConfig struct {
	// URL is the URL for the NATS server
	URL string `mapstructure:"url" default:"nats://localhost:4222"`
	// Prefix is the prefix for the NATS subjects to subscribe to
	Prefix string `mapstructure:"prefix" default:"minder"`
	// Queue is the name of the queue group to join when consuming messages
	// queue groups allow multiple process to round-robin process messages.
	Queue string `mapstructure:"queue" default:"minder"`
}
