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

package handlers

import (
	"encoding/json"
	"errors"
	"testing"

	watermill "github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/reflect/protoreflect"

	df "github.com/mindersec/minder/database/mock/fixtures"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/models"
	"github.com/mindersec/minder/internal/entities/properties"
	"github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/entities/properties/service/mock/fixtures"
	"github.com/mindersec/minder/internal/events"
	stubeventer "github.com/mindersec/minder/internal/events/stubs"
	mockgithub "github.com/mindersec/minder/internal/providers/github/mock"
	ghprops "github.com/mindersec/minder/internal/providers/github/properties"
	"github.com/mindersec/minder/internal/providers/manager"
	mock_manager "github.com/mindersec/minder/internal/providers/manager/mock"
	provManFixtures "github.com/mindersec/minder/internal/providers/manager/mock/fixtures"
	"github.com/mindersec/minder/internal/reconcilers/messages"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

var (
	projectID     = uuid.New()
	providerID    = uuid.New()
	repoID        = uuid.New()
	pullRequestID = uuid.New()

	repoName = "testorg/testrepo"
	pullName = "testorg/testrepo/789"

	repoEwp = models.EntityWithProperties{
		Entity: models.EntityInstance{
			ID:         repoID,
			Type:       minderv1.Entity_ENTITY_REPOSITORIES,
			Name:       repoName,
			ProviderID: providerID,
			ProjectID:  projectID,
		},
	}

	repoPropHookID = int64(456)

	repoPropMap = map[string]any{
		properties.PropertyName:          repoName,
		ghprops.RepoPropertyName:         "testrepo",
		ghprops.RepoPropertyOwner:        "testorg",
		ghprops.RepoPropertyId:           int64(123),
		ghprops.RepoPropertyHookId:       repoPropHookID,
		properties.RepoPropertyIsPrivate: false,
		properties.RepoPropertyIsFork:    false,
	}

	pullRequestEwp = models.EntityWithProperties{
		Entity: models.EntityInstance{
			ID:         pullRequestID,
			Type:       minderv1.Entity_ENTITY_PULL_REQUESTS,
			Name:       pullName,
			ProviderID: providerID,
			ProjectID:  projectID,
		},
	}
	pullRequestPropMap = map[string]any{
		properties.PropertyName:    pullName,
		ghprops.PullPropertyNumber: int64(789),
	}

	githubHint = service.ByUpstreamHint{
		ProviderImplements: db.NullProviderType{
			ProviderType: db.ProviderTypeGithub,
			Valid:        true,
		},
	}
)

type (
	providerMock        = *mockgithub.MockGitHub
	providerMockBuilder = func(controller *gomock.Controller) providerMock
)

func getPullRequestProperties() *properties.Properties {
	props, err := properties.NewProperties(pullRequestPropMap)
	if err != nil {
		panic(err)
	}

	return props
}

func newProviderMock(opts ...func(providerMock)) providerMockBuilder {
	return func(ctrl *gomock.Controller) providerMock {
		mock := mockgithub.NewMockGitHub(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func withSuccessfulGetEntityName(name string) func(providerMock) {
	return func(mock providerMock) {
		mock.EXPECT().
			GetEntityName(gomock.Any(), gomock.Any()).
			Return(name, nil)
	}
}

func withSuccessfulFetchAllProperties(props *properties.Properties) func(mock providerMock) {
	return func(mock providerMock) {
		mock.EXPECT().
			FetchAllProperties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(props, nil)
	}
}

func WithSuccessfulPropertiesToProtoMessage(proto protoreflect.ProtoMessage) func(mock providerMock) {
	return func(mock providerMock) {
		mock.EXPECT().
			PropertiesToProtoMessage(gomock.Any(), gomock.Any()).
			Return(proto, nil)
	}
}

func buildEwp(t *testing.T, ewp models.EntityWithProperties, propMap map[string]any) *models.EntityWithProperties {
	t.Helper()

	entProps, err := properties.NewProperties(propMap)
	require.NoError(t, err)
	ewp.Properties = entProps

	return &ewp
}

func checkRepoEntityMessage(t *testing.T, msg *watermill.Message) {
	t.Helper()

	var event messages.MinderEvent

	err := json.Unmarshal(msg.Payload, &event)
	require.NoError(t, err)
	require.Equal(t, minderv1.Entity_ENTITY_REPOSITORIES, event.EntityType)
	require.Equal(t, repoID, event.EntityID)
}

func checkRepoMessage(t *testing.T, msg *watermill.Message) {
	t.Helper()

	eiw, err := entities.ParseEntityEvent(msg)
	require.NoError(t, err)
	require.NotNil(t, eiw)

	pbrepo, ok := eiw.Entity.(*minderv1.Repository)
	require.True(t, ok)
	assert.Equal(t, repoPropMap[ghprops.RepoPropertyName].(string), pbrepo.Name)
	assert.Equal(t, repoPropMap[ghprops.RepoPropertyOwner].(string), pbrepo.Owner)
	assert.Equal(t, repoPropMap[ghprops.RepoPropertyId].(int64), pbrepo.RepoId)
	assert.Equal(t, repoPropMap[properties.RepoPropertyIsFork].(bool), pbrepo.IsFork)
}

func checkPullRequestMessage(t *testing.T, msg *watermill.Message) {
	t.Helper()

	eiw, err := entities.ParseEntityEvent(msg)
	require.NoError(t, err)
	require.NotNil(t, eiw)

	pbpr, ok := eiw.Entity.(*minderv1.PullRequest)
	require.True(t, ok)
	assert.Equal(t, pullRequestPropMap[ghprops.PullPropertyNumber].(int64), pbpr.Number)
}

type handlerBuilder func(
	evt events.Publisher,
	store db.Store,
	propSvc service.PropertiesService,
	provMgr manager.ProviderManager,
) events.Consumer

func refreshEntityHandlerBuilder(
	evt events.Publisher,
	store db.Store,
	propSvc service.PropertiesService,
	provMgr manager.ProviderManager,
) events.Consumer {
	return NewRefreshEntityAndEvaluateHandler(evt, store, propSvc, provMgr)
}

func refreshByIDHandlerBuilder(
	evt events.Publisher,
	store db.Store,
	propSvc service.PropertiesService,
	provMgr manager.ProviderManager,
) events.Consumer {
	return NewRefreshByIDAndEvaluateHandler(evt, store, propSvc, provMgr)
}

func addOriginatingEntityHandlerBuilder(
	evt events.Publisher,
	store db.Store,
	propSvc service.PropertiesService,
	provMgr manager.ProviderManager,
) events.Consumer {
	return NewAddOriginatingEntityHandler(evt, store, propSvc, provMgr)
}

func removeOriginatingEntityHandlerBuilder(
	evt events.Publisher,
	store db.Store,
	propSvc service.PropertiesService,
	provMgr manager.ProviderManager,
) events.Consumer {
	return NewRemoveOriginatingEntityHandler(evt, store, propSvc, provMgr)
}

func getAndDeleteEntityHandlerBuilder(
	evt events.Publisher,
	store db.Store,
	propSvc service.PropertiesService,
	_ manager.ProviderManager,
) events.Consumer {
	return NewGetEntityAndDeleteHandler(evt, store, propSvc)
}

func TestRefreshEntityAndDoHandler_HandleRefreshEntityAndEval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		messageBuilder       func() *message.HandleEntityAndDoMessage
		setupPropSvcMocks    func() fixtures.MockPropertyServiceBuilder
		mockStoreFunc        df.MockStoreBuilder
		providerManagerSetup func(prov provifv1.Provider) provManFixtures.ProviderManagerMockBuilder
		providerSetup        providerMockBuilder
		expectedPublish      bool
		topic                string
		checkWmMsg           func(t *testing.T, msg *watermill.Message)
		handlerBuilderFn     handlerBuilder
	}{
		{
			name:             "NewRefreshByIDAndEvaluateHandler: successful refresh and publish of an entity",
			handlerBuilderFn: refreshByIDHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				return message.NewEntityRefreshAndDoMessage().
					WithEntityID(repoID)
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				ewp := buildEwp(t, repoEwp, repoPropMap)
				protoEnt, err := ghprops.RepoV1FromProperties(ewp.Properties)
				require.NoError(t, err)

				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityWithPropertiesByID(repoID, ewp),
					fixtures.WithSuccessfulRetrieveAllPropertiesForEntity(),
					fixtures.WithSuccessfulEntityWithPropertiesAsProto(protoEnt),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
			),
			expectedPublish: true,
			topic:           events.TopicQueueEntityEvaluate,
			checkWmMsg:      checkRepoMessage,
		},
		{
			name:             "NewRefreshByIDAndEvaluateHandler: nil UUID does not publish",
			handlerBuilderFn: refreshByIDHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				return message.NewEntityRefreshAndDoMessage().
					WithEntityID(uuid.Nil)
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				return fixtures.NewMockPropertiesService()
			},
			mockStoreFunc:   df.NewMockStore(),
			expectedPublish: false,
			topic:           events.TopicQueueEntityEvaluate,
			checkWmMsg:      checkRepoMessage,
		},
		{
			name:             "NewRefreshEntityAndEvaluateHandler: successful refresh and publish of a repo",
			handlerBuilderFn: refreshEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				ewp := buildEwp(t, repoEwp, repoPropMap)
				protoEnt, err := ghprops.RepoV1FromProperties(ewp.Properties)
				require.NoError(t, err)

				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(ewp, githubHint),
					fixtures.WithSuccessfulRetrieveAllPropertiesForEntity(),
					fixtures.WithSuccessfulEntityWithPropertiesAsProto(protoEnt),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
			),
			expectedPublish: true,
			topic:           events.TopicQueueEntityEvaluate,
			checkWmMsg:      checkRepoMessage,
		},
		{
			name:             "NewRefreshEntityAndEvaluateHandler: if match_props match, publish",
			handlerBuilderFn: refreshEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				matchProps, _ := properties.NewProperties(map[string]any{
					ghprops.RepoPropertyHookId: repoPropHookID,
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithMatchProps(matchProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				ewp := buildEwp(t, repoEwp, repoPropMap)
				protoEnt, err := ghprops.RepoV1FromProperties(ewp.Properties)
				require.NoError(t, err)

				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(ewp, githubHint),
					fixtures.WithSuccessfulRetrieveAllPropertiesForEntity(),
					fixtures.WithSuccessfulEntityWithPropertiesAsProto(protoEnt),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
			),
			expectedPublish: true,
			topic:           events.TopicQueueEntityEvaluate,
			checkWmMsg:      checkRepoMessage,
		},
		{
			name:             "NewRefreshEntityAndEvaluateHandler: if match_props don't match, don't publish",
			handlerBuilderFn: refreshEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				matchProps, _ := properties.NewProperties(map[string]any{
					ghprops.RepoPropertyHookId: repoPropHookID + 1,
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithMatchProps(matchProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				ewp := buildEwp(t, repoEwp, repoPropMap)

				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(ewp, githubHint),
					fixtures.WithSuccessfulRetrieveAllPropertiesForEntity(),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
			),
			expectedPublish: false,
		},
		{
			name:             "NewRefreshEntityAndEvaluateHandler: private repo publishes if feature enabled",
			handlerBuilderFn: refreshEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				privateRepoMap := maps.Clone(repoPropMap)
				privateRepoMap[properties.RepoPropertyIsPrivate] = true

				ewp := buildEwp(t, repoEwp, privateRepoMap)
				protoEnt, err := ghprops.RepoV1FromProperties(ewp.Properties)
				require.NoError(t, err)

				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(ewp, githubHint),
					fixtures.WithSuccessfulRetrieveAllPropertiesForEntity(),
					fixtures.WithSuccessfulEntityWithPropertiesAsProto(protoEnt),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
				df.WithSuccessfulGetFeatureInProject(true),
			),
			expectedPublish: true,
			topic:           events.TopicQueueEntityEvaluate,
			checkWmMsg:      checkRepoMessage,
		},
		{
			name:             "NewRefreshEntityAndEvaluateHandler: private repo does not publish if feature disabled",
			handlerBuilderFn: refreshEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				privateRepoMap := maps.Clone(repoPropMap)
				privateRepoMap[properties.RepoPropertyIsPrivate] = true

				ewp := buildEwp(t, repoEwp, privateRepoMap)

				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(ewp, githubHint),
					fixtures.WithSuccessfulRetrieveAllPropertiesForEntity(),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
				df.WithSuccessfulGetFeatureInProject(false),
			),
			expectedPublish: false,
		},
		{
			name:             "NewRefreshEntityAndEvaluateHandler: archived repo does not publish",
			handlerBuilderFn: refreshEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				archivedRepoMap := maps.Clone(repoPropMap)
				archivedRepoMap[properties.RepoPropertyIsArchived] = true

				ewp := buildEwp(t, repoEwp, archivedRepoMap)

				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(ewp, githubHint),
					fixtures.WithSuccessfulRetrieveAllPropertiesForEntity(),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
			),
			expectedPublish: false,
		},
		{
			name:             "NewRefreshEntityAndEvaluateHandler: Failure to get an entity doesn't publish",
			handlerBuilderFn: refreshEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				return fixtures.NewMockPropertiesService(
					fixtures.WithFailedEntityByUpstreamHint(service.ErrEntityNotFound),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithRollbackTransaction(),
			),
			expectedPublish: false,
		},
		{
			name:             "NewRefreshEntityAndEvaluateHandler: Failure to retrieve all properties doesn't publish",
			handlerBuilderFn: refreshEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(&repoEwp, githubHint),
					fixtures.WithFailedRetrieveAllPropertiesForEntity(service.ErrEntityNotFound),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithRollbackTransaction(),
			),
			expectedPublish: false,
		},
		{
			name:             "NewRefreshEntityAndEvaluateHandler: Failure to convert entity to proto doesn't publish",
			handlerBuilderFn: refreshEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(&repoEwp, githubHint),
					fixtures.WithSuccessfulRetrieveAllPropertiesForEntity(),
					fixtures.WithFailedEntityWithPropertiesAsProto(errors.New("fart")),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
			),
			expectedPublish: false,
		},
		{
			name:             "NewAddOriginatingEntityHandler: Adding a pull request originating entity publishes",
			handlerBuilderFn: addOriginatingEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				prProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "789",
					ghprops.PullPropertyNumber:    int64(789),
				})
				originatorProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_PULL_REQUESTS, prProps).
					WithOriginator(minderv1.Entity_ENTITY_REPOSITORIES, originatorProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				pullEwp := buildEwp(t, pullRequestEwp, pullRequestPropMap)
				pullProtoEnt, err := ghprops.PullRequestV1FromProperties(pullEwp.Properties)
				require.NoError(t, err)

				repoPropsEwp := buildEwp(t, repoEwp, pullRequestPropMap)

				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(repoPropsEwp, githubHint),
					fixtures.WithSuccessfulEntityWithPropertiesAsProto(pullProtoEnt),
					fixtures.WithSuccessfulSaveAllProperties(),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
				df.WithSuccessfulUpsertPullRequestWithParams(
					db.PullRequest{ID: pullRequestID},
					db.EntityInstance{
						ID:         pullRequestID,
						EntityType: db.EntitiesPullRequest,
						Name:       "",
						ProjectID:  projectID,
						ProviderID: providerID,
						OriginatedFrom: uuid.NullUUID{
							UUID:  repoID,
							Valid: true,
						},
					},
					db.CreateOrEnsureEntityByIDParams{
						ID:         uuid.New(),
						EntityType: db.EntitiesPullRequest,
						Name:       pullName,
						ProjectID:  projectID,
						ProviderID: providerID,
						OriginatedFrom: uuid.NullUUID{
							UUID:  repoID,
							Valid: true,
						},
					},
				),
			),
			providerSetup: newProviderMock(
				withSuccessfulGetEntityName(pullName),
				withSuccessfulFetchAllProperties(getPullRequestProperties()),
				WithSuccessfulPropertiesToProtoMessage(&minderv1.PullRequest{
					Number: 789,
				}),
			),
			providerManagerSetup: func(prov provifv1.Provider) provManFixtures.ProviderManagerMockBuilder {
				return provManFixtures.NewProviderManagerMock(
					provManFixtures.WithSuccessfulInstantiateFromID(prov),
				)
			},
			expectedPublish: true,
			topic:           events.TopicQueueEntityEvaluate,
			checkWmMsg:      checkPullRequestMessage,
		},
		{
			name:             "NewRemoveOriginatingEntityHandler: Happy path does not publish",
			handlerBuilderFn: removeOriginatingEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				prProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "789",
					ghprops.PullPropertyNumber:    int64(789),
				})
				originatorProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_PULL_REQUESTS, prProps).
					WithOriginator(minderv1.Entity_ENTITY_REPOSITORIES, originatorProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				repoPropsEwp := buildEwp(t, repoEwp, repoPropMap)
				pullPropsEwp := buildEwp(t, pullRequestEwp, pullRequestPropMap)

				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(repoPropsEwp, githubHint),
					fixtures.WithSuccessfulEntityByUpstreamHint(pullPropsEwp, githubHint),
				)
			},
			mockStoreFunc: df.NewMockStore(
				df.WithTransaction(),
				df.WithSuccessfulDeleteEntity(pullRequestID, projectID),
			),
			providerSetup: newProviderMock(),
			providerManagerSetup: func(_ provifv1.Provider) provManFixtures.ProviderManagerMockBuilder {
				return provManFixtures.NewProviderManagerMock()
			},
			expectedPublish: false,
		},
		{
			name:             "NewGetEntityAndDeleteHandler: happy path publishes",
			handlerBuilderFn: getAndDeleteEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				repoPropsEwp := buildEwp(t, repoEwp, repoPropMap)

				return fixtures.NewMockPropertiesService(
					fixtures.WithSuccessfulEntityByUpstreamHint(repoPropsEwp, githubHint),
				)
			},
			mockStoreFunc: df.NewMockStore(),
			providerSetup: newProviderMock(),
			providerManagerSetup: func(_ provifv1.Provider) provManFixtures.ProviderManagerMockBuilder {
				return provManFixtures.NewProviderManagerMock()
			},
			expectedPublish: true,
			checkWmMsg:      checkRepoEntityMessage,
			topic:           events.TopicQueueReconcileEntityDelete,
		},
		{
			name:             "NewGetEntityAndDeleteHandler: failure to get entity does not publish",
			handlerBuilderFn: getAndDeleteEntityHandlerBuilder,
			messageBuilder: func() *message.HandleEntityAndDoMessage {
				getByProps, _ := properties.NewProperties(map[string]any{
					properties.PropertyUpstreamID: "123",
				})

				return message.NewEntityRefreshAndDoMessage().
					WithEntity(minderv1.Entity_ENTITY_REPOSITORIES, getByProps).
					WithProviderImplementsHint("github")
			},
			setupPropSvcMocks: func() fixtures.MockPropertyServiceBuilder {
				return fixtures.NewMockPropertiesService(
					fixtures.WithFailedEntityByUpstreamHint(service.ErrEntityNotFound),
				)
			},
			mockStoreFunc: df.NewMockStore(),
			providerSetup: newProviderMock(),
			providerManagerSetup: func(_ provifv1.Provider) provManFixtures.ProviderManagerMockBuilder {
				return provManFixtures.NewProviderManagerMock()
			},
			expectedPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			entityMsg := tt.messageBuilder()
			require.NotNil(t, entityMsg)

			handlerMsg := watermill.NewMessage(uuid.New().String(), nil)
			err := entityMsg.ToMessage(handlerMsg)
			require.NoError(t, err)

			mockPropSvc := tt.setupPropSvcMocks()(ctrl)
			mockStore := tt.mockStoreFunc(ctrl)
			stubEventer := &stubeventer.StubEventer{}

			var prov provifv1.Provider
			if tt.providerSetup != nil {
				prov = tt.providerSetup(ctrl)
			}

			var provMgr manager.ProviderManager
			if tt.providerManagerSetup != nil {
				provMgr = tt.providerManagerSetup(prov)(ctrl)
			} else {
				provMgr = mock_manager.NewMockProviderManager(ctrl)
			}

			handler := tt.handlerBuilderFn(stubEventer, mockStore, mockPropSvc, provMgr)
			refreshHandlerStruct, ok := handler.(*handleEntityAndDoBase)
			require.True(t, ok)
			err = refreshHandlerStruct.handleRefreshEntityAndDo(handlerMsg)
			assert.NoError(t, err)

			if !tt.expectedPublish {
				assert.Equal(t, 0, len(stubEventer.Sent), "Expected no publish calls")
				return
			}

			assert.Equal(t, 1, len(stubEventer.Topics), "Expected one topic")
			assert.Equal(t, tt.topic, stubEventer.Topics[0], "Expected topic to be %s", tt.topic)
			assert.Equal(t, 1, len(stubEventer.Sent), "Expected one publish call")
			tt.checkWmMsg(t, stubEventer.Sent[0])
		})
	}
}
