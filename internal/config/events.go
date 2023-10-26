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

package config

import "time"

// EventConfig is the configuration for mediator's eventing system.
type EventConfig struct {
	// Driver is the driver used to store events
	Driver string `mapstructure:"driver" default:"go-channel"`
	// RouterCloseTimeout is the timeout for closing the router
	RouterCloseTimeout time.Duration `mapstructure:"router_close_timeout" default:"10s"`
	// GoChannel is the configuration for the go channel event driver
	GoChannel *GoChannelEventConfig `mapstructure:"go-channel"`
}

// GoChannelEventConfig is the configuration for the go channel event driver
// for mediator's eventing system.
type GoChannelEventConfig struct {
	// BufferSize is the size of the buffer for the go channel
	BufferSize int64 `mapstructure:"buffer_size" default:"0"`
}
