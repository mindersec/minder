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

package events

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// dontNack is a middleware that will prevent the message from being nacked
// Note that it'll only use this behavior if we're using the SQL driver
func dontNack(driver string, l watermill.LoggerAdapter) func(h message.HandlerFunc) message.HandlerFunc {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			if driver != SQLDriver {
				return h(msg)
			}

			_, err := h(msg)

			msgID := msg.UUID
			l.Error("Clearing NACK from message so we don't infinitely retry", err, watermill.LogFields{
				"message_id": msgID,
			})
			return nil, nil
		}
	}
}
