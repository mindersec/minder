// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"time"

	"github.com/mindersec/minder/pkg/config"
)

// EventConfig is the configuration for minder's eventing system.
type EventConfig struct {
	// Driver is the driver used to store events
	Driver string `mapstructure:"driver" default:"go-channel"`
	// Flags is the configuration for selecting multiple publishing drivers
	// via the "alternate_message_driver" feature flag.
	Flags FlagDriverConfig `mapstructure:"flags"`
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

// FlagDriverConfig holds the configuration for selecting multiple publishing drivers
// when using feature flags to migrate from one publishing mechanism to another.
// When using the "flagged" driver, events will be read from _both_ drivers, but
// published to the driver selected by the flag configuration.
type FlagDriverConfig struct {
	// MainDriver is the default driver used to publish events if not selected
	// by the feature flag.
	MainDriver string `mapstructure:"main_driver" default:""`
	// AlternateDriver is a driver used to publish events selected by the
	// feature flag.
	AlternateDriver string `mapstructure:"alternate_driver" default:""`
}
