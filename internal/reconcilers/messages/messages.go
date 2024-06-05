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

// Package messages contains messages structs and builders for events
// handled by reconcilers.
package messages

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// RepoReconcilerEvent is an event that is sent to the reconciler topic
type RepoReconcilerEvent struct {
	// Project is the project that the event is relevant to
	Project uuid.UUID `json:"project"`
	// Repository is the repository to be reconciled
	Repository int64 `json:"repository" validate:"gte=0"`
}

// NewRepoReconcilerMessage creates a new repos init event
func NewRepoReconcilerMessage(providerID uuid.UUID, repoID int64, projectID uuid.UUID) (*message.Message, error) {
	evt := &RepoReconcilerEvent{
		Repository: repoID,
		Project:    projectID,
	}

	evtStr, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("error marshalling init event: %w", err)
	}

	msg := message.NewMessage(uuid.New().String(), evtStr)
	msg.Metadata.Set("provider_id", providerID.String())
	return msg, nil
}

// RepoEvent is an event that is sent when a new repository is
// created in an active installation.
//
// This struct is meant to be used with providers that can push events
// to Minder, or with providers that Minder can poll, and the code
// path it belongs to assumes that the repository was not previously
// registered.
type RepoEvent struct {
	ProviderID uuid.UUID `json:"provider_id" validate:"required"`
	ProjectID  uuid.UUID `json:"project_id" validate:"required"`
	RepoID     uuid.UUID `json:"repo_id"`
	RepoName   string    `json:"repo_name"`
	RepoOwner  string    `json:"repo_owner"`
}

// NewRepoEvent creates a new repo added event.
func NewRepoEvent() *RepoEvent {
	return &RepoEvent{}
}

// WithProviderID adds provider id to RepoEvent
func (e *RepoEvent) WithProviderID(providerID uuid.UUID) *RepoEvent {
	e.ProviderID = providerID
	return e
}

// WithProjectID adds project id to RepoEvent
func (e *RepoEvent) WithProjectID(projectID uuid.UUID) *RepoEvent {
	e.ProjectID = projectID
	return e
}

// WithRepoID adds project id to RepoEvent
func (e *RepoEvent) WithRepoID(repoID uuid.UUID) *RepoEvent {
	e.RepoID = repoID
	return e
}

// WithRepoName adds repository name to RepoEvent
func (e *RepoEvent) WithRepoName(repoName string) *RepoEvent {
	e.RepoName = repoName
	return e
}

// WithRepoOwner adds repository owner to RepoEvent
func (e *RepoEvent) WithRepoOwner(repoOwner string) *RepoEvent {
	e.RepoOwner = repoOwner
	return e
}

// ToMessage implements an interface that is currently used on the
// webhook handler. Such interface works by modifiying an existing
// message by means of side effect, which is unnecessary for this
// struct, thus its simplicity.
func (e *RepoEvent) ToMessage(msg *message.Message) error {
	payload, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("error marshalling event: %w", err)
	}

	msg.Payload = payload

	return nil
}
