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
	"context"
	"log"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/stacklok/mediator/internal/events"
	"github.com/stacklok/mediator/internal/reconcilers"
	"github.com/stacklok/mediator/pkg/db"
)

type sampleHandler struct {
	// This is a sample field that does nothing; this could be (for example)
	// a github client API handle
	store db.Store
}

// Register implements the Consumer interface.
func (s *sampleHandler) Register(r events.Registrar) {
	r.Register("security_and_analysis", s.handleSecurityAndAnalysisEvent)
	r.Register("branch_protection_rule", s.handleBranchProtectionEventGithub)
}

// NewHandler acts as a constructor for the sampleHandler.
func NewHandler(store db.Store) events.Consumer {
	return &sampleHandler{
		store: store,
	}
}

func (s *sampleHandler) handleSecurityAndAnalysisEvent(msg *message.Message) error {
	err := reconcilers.ParseSecretScanningEventGithub(context.Background(), s.store, msg)
	if err != nil {
		log.Printf("error parsing secret scanning event: %v", err)
		return err
	}
	return nil
}

func (s *sampleHandler) handleBranchProtectionEventGithub(msg *message.Message) error {
	err := reconcilers.ParseBranchProtectionEventGithub(context.Background(), s.store, msg)
	if err != nil {
		return err
	}
	return nil
}
