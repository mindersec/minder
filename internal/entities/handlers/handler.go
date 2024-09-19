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

// Package handlers contains the message handlers for entities.
package handlers

import (
	watermill "github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/entities/handlers/message"
	"github.com/stacklok/minder/internal/entities/handlers/strategies"
	entStrategies "github.com/stacklok/minder/internal/entities/handlers/strategies/entity"
	msgStrategies "github.com/stacklok/minder/internal/entities/handlers/strategies/message"
	propertyService "github.com/stacklok/minder/internal/entities/properties/service"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers/manager"
)

type handleEntityAndDoBase struct {
	evt events.Publisher

	refreshEntity strategies.GetEntityStrategy
	createMessage strategies.MessageCreateStrategy

	handlerName        string
	forwardHandlerName string

	handlerMiddleware []watermill.HandlerMiddleware
}

// Register satisfies the events.Consumer interface.
func (b *handleEntityAndDoBase) Register(r events.Registrar) {
	r.Register(b.handlerName, b.handleRefreshEntityAndDo, b.handlerMiddleware...)
}

// handleRefreshEntityAndDo handles the refresh entity and forwarding a new message to the
// next handler. Creating the message and the way the entity is refreshed is determined by the
// strategies passed in.
//
// The handler doesn't retry on errors, it just logs them. We've had issues with retrying
// recently and it's unclear if there are any errors we /can/ retry on. We should identify
// errors to retry on and implement that in the future.
func (b *handleEntityAndDoBase) handleRefreshEntityAndDo(msg *watermill.Message) error {
	ctx := msg.Context()

	l := zerolog.Ctx(ctx).With().
		Str("messageStrategy", b.createMessage.GetName()).
		Str("refreshStrategy", b.refreshEntity.GetName()).
		Logger()

	// unmarshal the message
	entMsg, err := message.ToEntityRefreshAndDo(msg)
	if err != nil {
		l.Error().Err(err).Msg("error unpacking message")
		return nil
	}
	l.Debug().Msg("message unpacked")

	// call refreshEntity
	ewp, err := b.refreshEntity.GetEntity(ctx, entMsg)
	if err != nil {
		l.Error().Err(err).Msg("error refreshing entity")
		// do not return error in the handler, just log it
		// we might want to special-case retrying /some/ errors specifically those from the
		// provider, but for now, just log it
		return nil
	}

	if ewp != nil {
		l.Debug().
			Str("entityID", ewp.Entity.ID.String()).
			Str("providerID", ewp.Entity.ProviderID.String()).
			Msg("entity refreshed")
	} else {
		l.Debug().Msg("entity not retrieved")
	}

	nextMsg, err := b.createMessage.CreateMessage(ctx, ewp)
	if err != nil {
		l.Error().Err(err).Msg("error creating message")
		return nil
	}

	// If nextMsg is nil, it means we don't need to publish anything (entity not found)
	if nextMsg != nil {
		l.Debug().Msg("publishing message")
		if err := b.evt.Publish(b.forwardHandlerName, nextMsg); err != nil {
			l.Error().Err(err).Msg("error publishing message")
			return nil
		}
	} else {
		l.Info().Msg("no message to publish")
	}

	return nil
}

// NewRefreshEntityAndEvaluateHandler creates a new handler that refreshes an entity and evaluates it.
func NewRefreshEntityAndEvaluateHandler(
	evt events.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	handlerMiddleware ...watermill.HandlerMiddleware,
) events.Consumer {
	return &handleEntityAndDoBase{
		evt: evt,

		refreshEntity: entStrategies.NewRefreshEntityByUpstreamPropsStrategy(propSvc, provMgr, store),
		createMessage: msgStrategies.NewToEntityInfoWrapper(store, propSvc, provMgr),

		handlerName:        events.TopicQueueRefreshEntityAndEvaluate,
		forwardHandlerName: events.TopicQueueEntityEvaluate,

		handlerMiddleware: handlerMiddleware,
	}
}

// NewGetEntityAndDeleteHandler creates a new handler that gets an entity and deletes it.
func NewGetEntityAndDeleteHandler(
	evt events.Publisher,
	propSvc propertyService.PropertiesService,
	handlerMiddleware ...watermill.HandlerMiddleware,
) events.Consumer {
	return &handleEntityAndDoBase{
		evt: evt,

		refreshEntity: entStrategies.NewGetEntityByUpstreamIDStrategy(propSvc),
		createMessage: msgStrategies.NewToMinderEntity(),

		handlerName:        events.TopicQueueGetEntityAndDelete,
		forwardHandlerName: events.TopicQueueReconcileEntityDelete,

		handlerMiddleware: handlerMiddleware,
	}
}

// NewAddOriginatingEntityHandler creates a new handler that adds an originating entity.
func NewAddOriginatingEntityHandler(
	evt events.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	handlerMiddleware ...watermill.HandlerMiddleware,
) events.Consumer {
	return &handleEntityAndDoBase{
		evt: evt,

		refreshEntity: entStrategies.NewAddOriginatingEntityStrategy(propSvc, provMgr, store),
		createMessage: msgStrategies.NewToEntityInfoWrapper(store, propSvc, provMgr),

		handlerName:        events.TopicQueueOriginatingEntityAdd,
		forwardHandlerName: events.TopicQueueEntityEvaluate,

		handlerMiddleware: handlerMiddleware,
	}
}

// NewRemoveOriginatingEntityHandler creates a new handler that removes an originating entity.
func NewRemoveOriginatingEntityHandler(
	evt events.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	handlerMiddleware ...watermill.HandlerMiddleware,
) events.Consumer {
	return &handleEntityAndDoBase{
		evt: evt,

		refreshEntity: entStrategies.NewDelOriginatingEntityStrategy(propSvc, provMgr, store),
		createMessage: msgStrategies.NewCreateEmpty(),

		handlerName: events.TopicQueueOriginatingEntityDelete,

		handlerMiddleware: handlerMiddleware,
	}
}
