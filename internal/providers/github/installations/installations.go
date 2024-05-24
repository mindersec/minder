//
// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package installations contains logic relating to GitHub provider installations
package installations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers/github/service"
)

const (
	// ProviderInstallationTopic is the topic for when a provider installation is removed
	ProviderInstallationTopic = "internal.provider.installation.removed.event"
)

// ProviderInstallationEvent is an event that occurs when a provider installation changes
type ProviderInstallationEvent string

const (
	// ProviderInstanceRemovedEvent is an event that occurs when a provider instance is removed
	ProviderInstanceRemovedEvent ProviderInstallationEvent = "provider_instance_removed"
)

const (
	// InstallationEventKey is the key for the event in the message metadata (e.g. removed)
	InstallationEventKey = "event"
	// ClassKey is the key for the provider class in the message metadata
	ClassKey = "class"
)

// InstallationManager is a struct representing the installation manager
type InstallationManager struct {
	svc service.GitHubProviderService
}

// NewInstallationManager creates a new installation manager
func NewInstallationManager(
	svc service.GitHubProviderService,
) *InstallationManager {
	return &InstallationManager{
		svc: svc,
	}
}

// Register implements the Consumer interface.
func (im *InstallationManager) Register(reg events.Registrar) {
	reg.Register(ProviderInstallationTopic, im.handleProviderInstallationEvent)
}

func (im *InstallationManager) handleProviderInstallationEvent(msg *message.Message) error {
	ctx := msg.Context()
	zerolog.Ctx(ctx).Info().Msg("Handling provider installation event")

	event := ProviderInstallationEvent(msg.Metadata.Get(InstallationEventKey))
	if event == ProviderInstanceRemovedEvent {
		return im.handleProviderInstanceRemovedEvent(ctx, msg)
	}
	zerolog.Ctx(ctx).Error().Msgf("Unknown event: %s", event)
	return nil
}

func (im *InstallationManager) handleProviderInstanceRemovedEvent(ctx context.Context, msg *message.Message) error {
	var payload service.GitHubAppInstallationDeletedPayload

	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return im.svc.DeleteGitHubAppInstallation(ctx, payload.InstallationID)
}

// InstallationInfoWrapper is a helper struct to gether information
// about installations from events.
// It's able to build a message.Message from the information it
// gathers.
type InstallationInfoWrapper struct {
	ProviderClass db.ProviderClass
	Payload       []byte
}

// NewInstallationInfoWrapper returns an empty
// *InstallationInfoWrapper for the caller to populate.
func NewInstallationInfoWrapper() *InstallationInfoWrapper {
	return &InstallationInfoWrapper{}
}

// WithProviderClass sets the provider class for this Installation
func (iiw *InstallationInfoWrapper) WithProviderClass(
	class db.ProviderClass,
) *InstallationInfoWrapper {
	iiw.ProviderClass = class
	return iiw
}

// WithPayload sets the payload for the installation.
//
// It does not perform any sort of validation on the payload, i.e. it
// coud be empty byte array, empty string, or even an invalid json.
func (iiw *InstallationInfoWrapper) WithPayload(
	payload []byte,
) *InstallationInfoWrapper {
	iiw.Payload = payload
	return iiw
}

// ToMessage sets the information to a message.Message. It works via
// side effect.
func (iiw *InstallationInfoWrapper) ToMessage(msg *message.Message) error {
	if iiw.ProviderClass == "" {
		return errors.New("provider class is required")
	}
	if len(iiw.Payload) == 0 {
		return errors.New("payload is empty")
	}

	msg.Metadata.Set(InstallationEventKey, string(ProviderInstanceRemovedEvent))
	msg.Metadata.Set(ClassKey, string(iiw.ProviderClass))
	msg.Payload = iiw.Payload

	return nil
}
