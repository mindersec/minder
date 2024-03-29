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

package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
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
	evt     events.Publisher
	svc     ProviderService
	querier db.Querier
}

// NewInstallationManager creates a new installation manager
func NewInstallationManager(
	evt events.Publisher,
	querier db.Querier,
	svc ProviderService,
) (*InstallationManager, error) {
	return &InstallationManager{
		evt:     evt,
		svc:     svc,
		querier: querier,
	}, nil
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
	return nil
}

func (im *InstallationManager) handleProviderInstanceRemovedEvent(ctx context.Context, msg *message.Message) error {
	class := msg.Metadata.Get(ClassKey)

	if db.ProviderClass(class) != db.ProviderClassGithubApp {
		zerolog.Ctx(ctx).Error().Str("class", class).Msg("Provider class is not supported")
		return nil
	}

	var payload GitHubAppInstallationDeletedPayload

	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return im.svc.DeleteGitHubAppInstallation(ctx, payload.InstallationID)
}

// ProviderInstanceRemovedMessage returns the provider installation event from the message
func ProviderInstanceRemovedMessage(
	msg *message.Message,
	providerClass db.ProviderClass,
	payload []byte,
) {
	msg.Metadata.Set(InstallationEventKey, string(ProviderInstanceRemovedEvent))
	msg.Metadata.Set(ClassKey, string(providerClass))
	msg.Payload = payload
}
