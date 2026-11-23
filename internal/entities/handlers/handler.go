// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package handlers contains the message handlers for entities.
package handlers

import (
	"context"
	"errors"
	"slices"

	watermill "github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/handlers/strategies"
	entStrategies "github.com/mindersec/minder/internal/entities/handlers/strategies/entity"
	msgStrategies "github.com/mindersec/minder/internal/entities/handlers/strategies/message"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	entitySvc "github.com/mindersec/minder/internal/entities/service"
	"github.com/mindersec/minder/internal/projects/features"
	"github.com/mindersec/minder/internal/providers/manager"
	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	"github.com/mindersec/minder/pkg/eventer/constants"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

var (
	errPrivateRepoNotAllowed  = errors.New("private repositories are not allowed in this project")
	errArchivedRepoNotAllowed = errors.New("archived repositories are not evaluated")
	errPropsDoNotMatch        = errors.New("properties do not match")
)

type handleEntityAndDoBase struct {
	evt   interfaces.Publisher
	store db.Store

	refreshEntity strategies.GetEntityStrategy
	createMessage strategies.MessageCreateStrategy

	handlerName        string
	forwardHandlerName string

	// upstreamGoneHandlerName, when set, receives a delete message for an
	// artifact whose upstream no longer exists.
	upstreamGoneHandlerName string

	handlerMiddleware []watermill.HandlerMiddleware
}

// Register satisfies the events.Consumer interface.
func (b *handleEntityAndDoBase) Register(r interfaces.Registrar) {
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
	if b.handleUpstreamGone(ctx, err) {
		return nil
	}
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

	forward, err := b.forwardEntityCheck(ctx, entMsg, ewp)
	if err != nil {
		l.Error().Err(err).Msg("error checking entity")
		return nil
	} else if !forward {
		return nil
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

// handleUpstreamGone publishes a delete for an artifact whose upstream tag no
// longer exists, and reports whether it did. Other entity types, and
// artifacts from providers outside deletesOnUpstreamNotFound, keep the
// existing log-and-drop behavior.
//
// The artifact is deleted on the first not-found, with no retry count or
// grace period. This keeps the logic simple and stops a deleted tag from
// being retried every reminder cycle, at the cost of deleting an artifact
// whose registry briefly returns 404, e.g. during a re-push or mirror sync.
// Such an artifact has to be registered again: manually for DockerHub and
// Quay, which send no push events. Revisit this if spurious deletes show up.
// deletesOnUpstreamNotFound lists the provider classes whose artifact
// not-found reliably means the tag is gone. GitHub is excluded because its
// packages API also returns 404 when the token loses access to a package.
var deletesOnUpstreamNotFound = []db.ProviderClass{
	db.ProviderClassDockerhub,
	db.ProviderClassQuay,
}

func (b *handleEntityAndDoBase) handleUpstreamGone(ctx context.Context, err error) bool {
	var notFound *strategies.UpstreamEntityNotFoundError
	if b.upstreamGoneHandlerName == "" || !errors.As(err, &notFound) ||
		notFound.Entity.Entity.Type != v1.Entity_ENTITY_ARTIFACTS {
		return false
	}

	l := zerolog.Ctx(ctx).With().
		Str("entityID", notFound.Entity.Entity.ID.String()).
		Str("projectID", notFound.Entity.Entity.ProjectID.String()).
		Logger()

	prov, err := b.store.GetProviderByID(ctx, notFound.Entity.Entity.ProviderID)
	if err != nil {
		l.Error().Err(err).Msg("error getting provider for artifact gone upstream")
		return false
	}
	if !slices.Contains(deletesOnUpstreamNotFound, prov.Class) {
		return false
	}

	msg, err := msgStrategies.NewToMinderEntity().CreateMessage(ctx, notFound.Entity)
	if err != nil {
		l.Error().Err(err).Msg("error creating delete message for artifact gone upstream")
		return true
	}
	msg.SetContext(ctx)

	if err := b.evt.Publish(b.upstreamGoneHandlerName, msg); err != nil {
		l.Error().Err(err).Msg("error publishing delete message for artifact gone upstream")
		return true
	}

	l.Info().Msg("artifact not found upstream, deleting entity")
	return true
}

func (b *handleEntityAndDoBase) forwardEntityCheck(
	ctx context.Context,
	entMsg *message.HandleEntityAndDoMessage,
	ewp *models.EntityWithProperties) (bool, error) {
	if ewp == nil {
		return true, nil
	}

	err := b.matchPropertiesCheck(entMsg, ewp)
	if errors.Is(err, errPropsDoNotMatch) {
		zerolog.Ctx(ctx).Debug().Err(err).Msg("properties do not match")
		return false, nil
	} else if err != nil {
		return false, err
	}

	err = b.repoPrivateOrArchivedCheck(ctx, ewp)
	if errors.Is(err, errPrivateRepoNotAllowed) || errors.Is(err, errArchivedRepoNotAllowed) {
		zerolog.Ctx(ctx).Debug().Err(err).Msg("private or archived repo")
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// matchPropertiesCheck checks if the properties of the entity match the properties in the message.
// this is different from the hint check, which is a check to see if the entity comes from where we expect
// it to come from. A concrete example is receiving a "meta" event on a webhook we don't manage, in that case
// we'd want to match on the webhook ID and only proceed if it matches with the webhook ID minder tracks.
func (*handleEntityAndDoBase) matchPropertiesCheck(
	entMsg *message.HandleEntityAndDoMessage,
	ewp *models.EntityWithProperties) error {
	// nothing to match against, so we're good
	if entMsg.MatchProps == nil {
		return nil
	}

	matchProps := properties.NewProperties(entMsg.MatchProps)

	for propName, prop := range matchProps.Iterate() {
		entProp := ewp.Properties.GetProperty(propName)
		if !prop.Equal(entProp) {
			return errPropsDoNotMatch
		}
	}

	return nil
}

func (b *handleEntityAndDoBase) repoPrivateOrArchivedCheck(
	ctx context.Context,
	ewp *models.EntityWithProperties) error {
	if ewp.Entity.Type == v1.Entity_ENTITY_REPOSITORIES &&
		ewp.Properties.GetProperty(properties.RepoPropertyIsPrivate).GetBool() &&
		!features.ProjectAllowsPrivateRepos(ctx, b.store, ewp.Entity.ProjectID) {
		return errPrivateRepoNotAllowed
	}

	if ewp.Entity.Type == v1.Entity_ENTITY_REPOSITORIES &&
		ewp.Properties.GetProperty(properties.RepoPropertyIsArchived).GetBool() {
		return errArchivedRepoNotAllowed
	}

	return nil
}

// NewRefreshByIDAndEvaluateHandler creates a new handler that refreshes an entity and evaluates it.
func NewRefreshByIDAndEvaluateHandler(
	evt interfaces.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	handlerMiddleware ...watermill.HandlerMiddleware,
) interfaces.Consumer {
	return &handleEntityAndDoBase{
		evt:   evt,
		store: store,

		refreshEntity: entStrategies.NewRefreshEntityByIDStrategy(propSvc, provMgr, store),
		createMessage: msgStrategies.NewToEntityInfoWrapper(store, propSvc, provMgr),

		handlerName:             constants.TopicQueueRefreshEntityByIDAndEvaluate,
		forwardHandlerName:      constants.TopicQueueEntityEvaluate,
		upstreamGoneHandlerName: constants.TopicQueueReconcileEntityDelete,

		handlerMiddleware: handlerMiddleware,
	}
}

// NewRefreshEntityAndEvaluateHandler creates a new handler that refreshes an entity and evaluates it.
func NewRefreshEntityAndEvaluateHandler(
	evt interfaces.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	handlerMiddleware ...watermill.HandlerMiddleware,
) interfaces.Consumer {
	return &handleEntityAndDoBase{
		evt:   evt,
		store: store,

		refreshEntity: entStrategies.NewRefreshEntityByUpstreamPropsStrategy(propSvc, provMgr, store),
		createMessage: msgStrategies.NewToEntityInfoWrapper(store, propSvc, provMgr),

		handlerName:        constants.TopicQueueRefreshEntityAndEvaluate,
		forwardHandlerName: constants.TopicQueueEntityEvaluate,

		handlerMiddleware: handlerMiddleware,
	}
}

// NewGetEntityAndDeleteHandler creates a new handler that gets an entity and deletes it.
func NewGetEntityAndDeleteHandler(
	evt interfaces.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	handlerMiddleware ...watermill.HandlerMiddleware,
) interfaces.Consumer {
	return &handleEntityAndDoBase{
		evt:   evt,
		store: store,

		refreshEntity: entStrategies.NewGetEntityByUpstreamIDStrategy(propSvc),
		createMessage: msgStrategies.NewToMinderEntity(),

		handlerName:        constants.TopicQueueGetEntityAndDelete,
		forwardHandlerName: constants.TopicQueueReconcileEntityDelete,

		handlerMiddleware: handlerMiddleware,
	}
}

// NewAddOriginatingEntityHandler creates a new handler that adds an originating entity.
func NewAddOriginatingEntityHandler(
	evt interfaces.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	entityCreator entitySvc.EntityCreator,
	handlerMiddleware ...watermill.HandlerMiddleware,
) interfaces.Consumer {
	return &handleEntityAndDoBase{
		evt:   evt,
		store: store,

		refreshEntity: entStrategies.NewAddOriginatingEntityStrategy(propSvc, provMgr, store, entityCreator),
		createMessage: msgStrategies.NewToEntityInfoWrapper(store, propSvc, provMgr),

		handlerName:        constants.TopicQueueOriginatingEntityAdd,
		forwardHandlerName: constants.TopicQueueEntityEvaluate,

		handlerMiddleware: handlerMiddleware,
	}
}

// NewRemoveOriginatingEntityHandler creates a new handler that removes an originating entity.
func NewRemoveOriginatingEntityHandler(
	evt interfaces.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	handlerMiddleware ...watermill.HandlerMiddleware,
) interfaces.Consumer {
	return &handleEntityAndDoBase{
		evt: evt,

		refreshEntity: entStrategies.NewDelOriginatingEntityStrategy(propSvc, provMgr, store),
		createMessage: msgStrategies.NewCreateEmpty(),

		handlerName: constants.TopicQueueOriginatingEntityDelete,

		handlerMiddleware: handlerMiddleware,
	}
}
