package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	propertyService "github.com/stacklok/minder/internal/entities/properties/service"
	"github.com/stacklok/minder/internal/events"
	ghprop "github.com/stacklok/minder/internal/providers/github/properties"
	"github.com/stacklok/minder/internal/providers/manager"
	"github.com/stacklok/minder/internal/reconcilers/messages"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type messageCreateStrategy interface {
	CreateMessage(
		ctx context.Context, ewp *models.EntityWithProperties,
	) (*message.Message, error)
	GetName() string
}

type toEntityInfoWrapper struct {
	store   db.Store
	propSvc propertyService.PropertiesService
	provMgr manager.ProviderManager
}

func newToEntityInfoWrapper(
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
) *toEntityInfoWrapper {
	return &toEntityInfoWrapper{
		store:   store,
		propSvc: propSvc,
		provMgr: provMgr,
	}
}

func (c *toEntityInfoWrapper) CreateMessage(
	ctx context.Context, ewp *models.EntityWithProperties,
) (*message.Message, error) {
	pbEnt, err := c.propSvc.EntityWithPropertiesAsProto(ctx, ewp, c.provMgr)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	m := message.NewMessage(uuid.New().String(), nil)

	eiw := entities.NewEntityInfoWrapper().
		WithProjectID(ewp.Entity.ProjectID).
		WithProviderID(ewp.Entity.ProviderID).
		WithProtoMessage(ewp.Entity.Type, pbEnt).
		WithID(ewp.Entity.Type, ewp.Entity.ID)

	// in case the entity originated from another entity, add that information as well.
	// the property service does not provide this information (should it?) so we need to fetch it from the store.
	// for now we could have hardcoded the entity type as everything originates from a repository,
	// but this is more flexible.
	if ewp.Entity.OriginatedFrom != uuid.Nil {
		dbEnt, err := c.store.GetEntityByID(ctx, ewp.Entity.OriginatedFrom)
		if err != nil {
			return nil, fmt.Errorf("error getting originating entity: %w", err)
		}
		eiw.WithID(entities.EntityTypeFromDB(dbEnt.EntityType), dbEnt.ID)
	}

	err = eiw.ToMessage(m)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to message: %w", err)
	}

	return m, nil
}

func (c *toEntityInfoWrapper) GetName() string {
	return "toEntityInfoWrapper"
}

type toMinderEntityStrategy struct{}

func (c *toMinderEntityStrategy) CreateMessage(_ context.Context, ewp *models.EntityWithProperties) (*message.Message, error) {
	m := message.NewMessage(uuid.New().String(), nil)

	entEvent := messages.NewMinderEvent().
		WithProjectID(ewp.Entity.ProjectID).
		WithProviderID(ewp.Entity.ProviderID).
		WithEntityType(ewp.Entity.Type.String()).
		WithEntityID(ewp.Entity.ID)

	err := entEvent.ToMessage(m)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to message: %w", err)
	}

	return m, nil
}

func (c *toMinderEntityStrategy) GetName() string {
	return "toMinderv1Entity"
}

type createEmpty struct{}

func (c *createEmpty) CreateMessage(_ context.Context, _ *models.EntityWithProperties) (*message.Message, error) {
	return nil, nil
}

func (c *createEmpty) GetName() string {
	return "empty"
}

func getEntityInner(
	ctx context.Context,
	entType minderv1.Entity,
	entPropMap map[string]any,
	hint EntityHint,
	propSvc propertyService.PropertiesService,
	getEntityOpts *propertyService.CallOptions,
) (*models.EntityWithProperties, error) {
	svcHint := propertyService.ByUpstreamIdHint{}
	if hint.ProviderHint != "" {
		svcHint.ProviderImplements.Valid = true
		if err := svcHint.ProviderImplements.ProviderType.Scan(hint.ProviderHint); err != nil {
			return nil, fmt.Errorf("error scanning provider type: %w", err)
		}
	}

	lookupProperties, err := properties.NewProperties(entPropMap)
	if err != nil {
		return nil, fmt.Errorf("error creating properties: %w", err)
	}

	upstreamID, err := lookupProperties.GetProperty(properties.PropertyUpstreamID).AsString()
	if err != nil {
		return nil, fmt.Errorf("error getting upstream ID: %w", err)
	}

	ewp, err := propSvc.EntityWithPropertiesByUpstreamID(
		ctx,
		entType,
		upstreamID,
		svcHint,
		getEntityOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("error searching entity by ID: %w", err)
	}

	return ewp, nil
}

type getEntityStrategy interface {
	GetEntity(
		ctx context.Context, entMsg *HandleEntityAndDoMessage,
	) (*models.EntityWithProperties, error)
	GetName() string
}

type getEntityByUpstreamIDStrategy struct {
	propSvc propertyService.PropertiesService
}

func newGetEntityByUpstreamIDStrategy(
	propSvc propertyService.PropertiesService,
) *getEntityByUpstreamIDStrategy {
	return &getEntityByUpstreamIDStrategy{
		propSvc: propSvc,
	}
}

func (g *getEntityByUpstreamIDStrategy) GetEntity(
	ctx context.Context, entMsg *HandleEntityAndDoMessage,
) (*models.EntityWithProperties, error) {
	return getEntityInner(ctx,
		entMsg.Entity.Type, entMsg.Entity.GetByProps, entMsg.Hint,
		g.propSvc,
		propertyService.CallBuilder())
}

func (g *getEntityByUpstreamIDStrategy) GetName() string {
	return "getEntityByUpstreamID"
}

type refreshEntityByUpstreamIDStrategy struct {
	propSvc propertyService.PropertiesService
	provMgr manager.ProviderManager
	store   db.Store
}

func newRefreshEntityByUpstreamIDStrategy(
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	store db.Store,
) *refreshEntityByUpstreamIDStrategy {
	return &refreshEntityByUpstreamIDStrategy{
		propSvc: propSvc,
		provMgr: provMgr,
		store:   store,
	}
}

func (r *refreshEntityByUpstreamIDStrategy) GetEntity(
	ctx context.Context, entMsg *HandleEntityAndDoMessage,
) (*models.EntityWithProperties, error) {
	getEnt, err := db.WithTransaction(r.store, func(t db.ExtendQuerier) (*models.EntityWithProperties, error) {
		ewp, err := getEntityInner(
			ctx,
			entMsg.Entity.Type, entMsg.Entity.GetByProps, entMsg.Hint,
			r.propSvc, propertyService.CallBuilder().WithStoreOrTransaction(t))
		if err != nil {
			return nil, fmt.Errorf("error getting entity: %w", err)
		}

		err = r.propSvc.RetrieveAllPropertiesForEntity(ctx, ewp, r.provMgr, propertyService.ReadBuilder())
		if err != nil {
			return nil, fmt.Errorf("error fetching repository: %w", err)
		}
		return ewp, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error refreshing entity: %w", err)
	}

	return getEnt, nil
}

func (r *refreshEntityByUpstreamIDStrategy) GetName() string {
	return "refreshEntityByUpstreamIDStrategy"
}

type addOriginatingEntityStrategy struct {
	propSvc propertyService.PropertiesService
	provMgr manager.ProviderManager
	store   db.Store
}

func newAddOriginatingEntityStrategy(
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	store db.Store,
) *addOriginatingEntityStrategy {
	return &addOriginatingEntityStrategy{
		propSvc: propSvc,
		provMgr: provMgr,
		store:   store,
	}
}

func (a *addOriginatingEntityStrategy) GetEntity(
	ctx context.Context, entMsg *HandleEntityAndDoMessage,
) (*models.EntityWithProperties, error) {
	childProps, err := properties.NewProperties(entMsg.Entity.GetByProps)
	if err != nil {
		return nil, fmt.Errorf("error creating properties: %w", err)
	}

	// store the originating entity
	childEwp, err := db.WithTransaction(a.store, func(t db.ExtendQuerier) (*models.EntityWithProperties, error) {
		parentEwp, err := getEntityInner(
			ctx,
			entMsg.Owner.Type, entMsg.Owner.GetByProps, entMsg.Hint,
			a.propSvc,
			propertyService.CallBuilder().WithStoreOrTransaction(t))
		if err != nil {
			return nil, fmt.Errorf("error getting parent entity: %w", err)
		}

		legacyId, err := a.upsertLegacyEntity(ctx, entMsg.Entity.Type, parentEwp, childProps, t)
		if err != nil {
			return nil, fmt.Errorf("error upserting legacy entity: %w", err)
		}

		prov, err := a.provMgr.InstantiateFromID(ctx, parentEwp.Entity.ProviderID)
		if err != nil {
			return nil, fmt.Errorf("error getting provider: %w", err)
		}

		childEntName, err := prov.GetEntityName(entMsg.Entity.Type, childProps)
		if err != nil {
			return nil, fmt.Errorf("error getting child entity name: %w", err)
		}

		childEnt, err := t.CreateOrEnsureEntityByID(ctx, db.CreateOrEnsureEntityByIDParams{
			ID:         legacyId,
			EntityType: entities.EntityTypeToDB(entMsg.Entity.Type),
			Name:       childEntName,
			ProjectID:  parentEwp.Entity.ProjectID,
			ProviderID: parentEwp.Entity.ProviderID,
			OriginatedFrom: uuid.NullUUID{
				UUID:  parentEwp.Entity.ID,
				Valid: true,
			},
		})
		if err != nil {
			return nil, err
		}

		upstreamProps, err := a.propSvc.RetrieveAllProperties(ctx, prov,
			parentEwp.Entity.ProjectID, parentEwp.Entity.ProviderID,
			childProps, entMsg.Entity.Type,
			propertyService.ReadBuilder().WithStoreOrTransaction(t),
		)
		if err != nil {
			return nil, fmt.Errorf("error retrieving properties: %w", err)
		}

		return models.NewEntityWithProperties(childEnt, upstreamProps), nil

	})

	if err != nil {
		return nil, fmt.Errorf("error storing originating entity: %w", err)
	}
	return childEwp, nil
}

func (a *addOriginatingEntityStrategy) GetName() string {
	return "addOriginatingEntityStrategy"
}

func (a *addOriginatingEntityStrategy) upsertLegacyEntity(
	ctx context.Context,
	entType minderv1.Entity,
	parentEwp *models.EntityWithProperties, childProps *properties.Properties,
	t db.ExtendQuerier,
) (uuid.UUID, error) {
	var legacyId uuid.UUID

	switch entType {
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		dbPr, err := t.UpsertPullRequest(ctx, db.UpsertPullRequestParams{
			RepositoryID: parentEwp.Entity.ID,
			PrNumber:     childProps.GetProperty(ghprop.PullPropertyNumber).GetInt64(),
		})
		if err != nil {
			return uuid.Nil, fmt.Errorf("error upserting pull request: %w", err)
		}
		legacyId = dbPr.ID
	case minderv1.Entity_ENTITY_ARTIFACTS:
		// TODO: remove this once we migrate artifacts to entities. We should get rid of the provider name.
		dbProv, err := t.GetProviderByID(ctx, parentEwp.Entity.ProviderID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("error getting provider: %w", err)
		}

		dbArtifact, err := t.UpsertArtifact(ctx, db.UpsertArtifactParams{
			RepositoryID: uuid.NullUUID{
				UUID:  parentEwp.Entity.ID,
				Valid: true,
			},
			ArtifactName:       childProps.GetProperty(ghprop.ArtifactPropertyName).GetString(),
			ArtifactType:       childProps.GetProperty(ghprop.ArtifactPropertyType).GetString(),
			ArtifactVisibility: childProps.GetProperty(ghprop.ArtifactPropertyVisibility).GetString(),
			ProjectID:          parentEwp.Entity.ProjectID,
			ProviderID:         parentEwp.Entity.ProviderID,
			ProviderName:       dbProv.Name,
		})
		if err != nil {
			return uuid.Nil, fmt.Errorf("error upserting artifact: %w", err)
		}
		legacyId = dbArtifact.ID
	}

	return legacyId, nil
}

type delOriginatingEntityStrategy struct {
	propSvc propertyService.PropertiesService
	provMgr manager.ProviderManager
	store   db.Store
}

func newDelOriginatingEntityStrategy(
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	store db.Store,
) *delOriginatingEntityStrategy {
	return &delOriginatingEntityStrategy{
		propSvc: propSvc,
		provMgr: provMgr,
		store:   store,
	}
}

func (d *delOriginatingEntityStrategy) GetEntity(
	ctx context.Context, entMsg *HandleEntityAndDoMessage,
) (*models.EntityWithProperties, error) {
	childProps, err := properties.NewProperties(entMsg.Entity.GetByProps)
	if err != nil {
		return nil, fmt.Errorf("error creating properties: %w", err)
	}

	tx, err := d.store.BeginTransaction()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		_ = d.store.Rollback(tx)
	}()

	txq := d.store.GetQuerierWithTransaction(tx)
	if txq == nil {
		return nil, fmt.Errorf("error getting querier")
	}

	parentEwp, err := getEntityInner(
		ctx,
		entMsg.Owner.Type, entMsg.Owner.GetByProps, entMsg.Hint,
		d.propSvc,
		propertyService.CallBuilder().WithStoreOrTransaction(txq))
	if err != nil {
		return nil, fmt.Errorf("error getting parent entity: %w", err)
	}

	prov, err := d.provMgr.InstantiateFromID(ctx, parentEwp.Entity.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("error getting provider: %w", err)
	}

	childEntName, err := prov.GetEntityName(entMsg.Entity.Type, childProps)
	if err != nil {
		return nil, fmt.Errorf("error getting child entity name: %w", err)
	}

	err = txq.DeleteEntityByName(ctx, db.DeleteEntityByNameParams{
		Name:      childEntName,
		ProjectID: parentEwp.Entity.ProjectID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if err := d.store.Commit(tx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return nil, nil
}

func (d *delOriginatingEntityStrategy) deleteLegacyEntity(
	ctx context.Context,
	entType minderv1.Entity,
	parentEwp *models.EntityWithProperties,
	childProps *properties.Properties,
	t db.ExtendQuerier,
) error {
	switch entType {
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		err := t.DeletePullRequest(ctx, db.DeletePullRequestParams{
			RepositoryID: parentEwp.Entity.ID,
			PrNumber:     childProps.GetProperty(ghprop.PullPropertyNumber).GetInt64(),
		})
		if err != nil {
			return fmt.Errorf("error deleting pull request: %w", err)
		}
	}

	return nil
}

func (d *delOriginatingEntityStrategy) GetName() string {
	return "delOriginatingEntityStrategy"
}

type handleEntityAndDoBase struct {
	evt events.Publisher

	refreshEntity getEntityStrategy
	createMessage messageCreateStrategy

	handlerName        string
	forwardHandlerName string

	handlerMiddleware []message.HandlerMiddleware
}

func (b *handleEntityAndDoBase) Register(r events.Registrar) {
	r.Register(b.handlerName, b.handleRefreshEntityAndDo, b.handlerMiddleware...)
}

func (b *handleEntityAndDoBase) handleRefreshEntityAndDo(msg *message.Message) error {
	ctx := msg.Context()

	l := zerolog.Ctx(ctx).With().
		Str("messageStrategy", b.createMessage.GetName()).
		Str("refreshStrategy", b.refreshEntity.GetName()).
		Logger()

	// unmarshal the message
	entMsg, err := messageToEntityRefreshAndDo(msg)
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

func NewRefreshEntityAndEvaluateHandler(
	evt events.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	handlerMiddleware ...message.HandlerMiddleware,
) events.Consumer {
	return &handleEntityAndDoBase{
		evt: evt,

		refreshEntity: newRefreshEntityByUpstreamIDStrategy(propSvc, provMgr, store),
		createMessage: newToEntityInfoWrapper(store, propSvc, provMgr),

		handlerName:        events.TopicQueueRefreshEntityAndEvaluate,
		forwardHandlerName: events.TopicQueueEntityEvaluate,

		handlerMiddleware: handlerMiddleware,
	}
}

func NewGetEntityAndDeleteHandler(
	evt events.Publisher,
	propSvc propertyService.PropertiesService,
	handlerMiddleware ...message.HandlerMiddleware,
) events.Consumer {
	return &handleEntityAndDoBase{
		evt: evt,

		refreshEntity: newGetEntityByUpstreamIDStrategy(propSvc),
		createMessage: &toMinderEntityStrategy{},

		handlerName:        events.TopicQueueGetEntityAndDelete,
		forwardHandlerName: events.TopicQueueReconcileEntityDelete,

		handlerMiddleware: handlerMiddleware,
	}
}

func NewAddOriginatingEntityHandler(
	evt events.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	handlerMiddleware ...message.HandlerMiddleware,
) events.Consumer {
	return &handleEntityAndDoBase{
		evt: evt,

		refreshEntity: newAddOriginatingEntityStrategy(propSvc, provMgr, store),
		createMessage: newToEntityInfoWrapper(store, propSvc, provMgr),

		handlerName:        events.TopicQueueOriginatingEntityAdd,
		forwardHandlerName: events.TopicQueueEntityEvaluate,

		handlerMiddleware: handlerMiddleware,
	}
}

func NewRemoveOriginatingEntityHandler(
	evt events.Publisher,
	store db.Store,
	propSvc propertyService.PropertiesService,
	provMgr manager.ProviderManager,
	handlerMiddleware ...message.HandlerMiddleware,
) events.Consumer {
	return &handleEntityAndDoBase{
		evt: evt,

		refreshEntity: newDelOriginatingEntityStrategy(propSvc, provMgr, store),
		createMessage: &createEmpty{},

		handlerName: events.TopicQueueOriginatingEntityDelete,

		handlerMiddleware: handlerMiddleware,
	}
}
