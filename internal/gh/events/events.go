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

// Package events provides implementations of all the event handlers
// the GitHub provider supports.
package events

import (
	"log"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stacklok/mediator/internal/events"
	"github.com/stacklok/mediator/pkg/db"
)

type sampleHandler struct {
	// This is a sample field that does nothing; this could be (for example)
	// a github client API handle
	store db.Store
}

// SampleHandler implements the Consumer interface
func (h *sampleHandler) Register(r events.Registrar) {
	r.Register("security_and_analysis", handleSecurityAndAnalysisEvent)
}

func NewHandler(store db.Store) events.Consumer {
	return &sampleHandler{
		store: store,
	}
}

func handleSecurityAndAnalysisEvent(msg *message.Message) error {
	log.Printf("Got a security_and_analysis event: %v", msg)
	return nil
}
