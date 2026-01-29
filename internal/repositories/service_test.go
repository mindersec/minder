// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	gh "github.com/google/go-github/v63/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/models"
	mock_propservice "github.com/mindersec/minder/internal/entities/properties/service/mock"
	mock_entityservice "github.com/mindersec/minder/internal/entities/service/mock"
	"github.com/mindersec/minder/internal/entities/service/validators"
	mockgithub "github.com/mindersec/minder/internal/providers/github/mock"
	ghprop "github.com/mindersec/minder/internal/providers/github/properties"
	"github.com/mindersec/minder/internal/providers/manager"
	pf "github.com/mindersec/minder/internal/providers/manager/mock/fixtures"
	"github.com/mindersec/minder/internal/repositories"
	"github.com/mindersec/minder/internal/util/ptr"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	mockevents "github.com/mindersec/minder/pkg/eventer/interfaces/mock"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// NOTE: Tests for CreateRepository that test the internal EntityCreator behavior have been
// moved to service_integration_test.go and internal/entities/service/entity_creator_test.go.
// The tests below now focus on the RepositoryService's direct responsibilities:
// - Calling EntityCreator with correct parameters
// - Converting the result to protobuf
// - Error propagation

func TestRepositoryService_CreateRepository(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		EntityCreator func(*mock_entityservice.MockEntityCreator)
		ServiceSetup  propSvcMockBuilder
		ExpectedError string
	}{
		{
			Name: "CreateRepository succeeds",
			EntityCreator: func(m *mock_entityservice.MockEntityCreator) {
				m.EXPECT().
					CreateEntity(gomock.Any(), gomock.Any(), projectID, pb.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any()).
					Return(&models.EntityWithProperties{
						Entity: models.EntityInstance{
							ID:         repoID,
							Type:       pb.Entity_ENTITY_REPOSITORIES,
							Name:       fmt.Sprintf("%s/%s", repoOwner, repoName),
							ProjectID:  projectID,
							ProviderID: uuid.UUID{},
						},
						Properties: publicProps,
					}, nil)
			},
			ServiceSetup: newPropSvcMock(withSucessfulEntityToProto),
		},
		{
			Name: "CreateRepository fails when EntityCreator fails",
			EntityCreator: func(m *mock_entityservice.MockEntityCreator) {
				m.EXPECT().
					CreateEntity(gomock.Any(), gomock.Any(), projectID, pb.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any()).
					Return(nil, errDefault)
			},
			ServiceSetup:  newPropSvcMock(),
			ExpectedError: "error creating repository",
		},
		{
			Name: "CreateRepository fails when proto conversion fails",
			EntityCreator: func(m *mock_entityservice.MockEntityCreator) {
				m.EXPECT().
					CreateEntity(gomock.Any(), gomock.Any(), projectID, pb.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any()).
					Return(&models.EntityWithProperties{
						Entity: models.EntityInstance{
							ID:         repoID,
							Type:       pb.Entity_ENTITY_REPOSITORIES,
							Name:       fmt.Sprintf("%s/%s", repoOwner, repoName),
							ProjectID:  projectID,
							ProviderID: uuid.UUID{},
						},
						Properties: publicProps,
					}, nil)
			},
			ServiceSetup:  newPropSvcMock(withFailedEntityToProto),
			ExpectedError: "error converting entity to protobuf",
		},
		{
			Name: "CreateRepository propagates private repo error",
			EntityCreator: func(m *mock_entityservice.MockEntityCreator) {
				m.EXPECT().
					CreateEntity(gomock.Any(), gomock.Any(), projectID, pb.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any()).
					Return(nil, validators.ErrPrivateRepoForbidden)
			},
			ServiceSetup:  newPropSvcMock(),
			ExpectedError: "private repositories are not allowed",
		},
		{
			Name: "CreateRepository propagates archived repo error",
			EntityCreator: func(m *mock_entityservice.MockEntityCreator) {
				m.EXPECT().
					CreateEntity(gomock.Any(), gomock.Any(), projectID, pb.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any()).
					Return(nil, validators.ErrArchivedRepoForbidden)
			},
			ServiceSetup:  newPropSvcMock(),
			ExpectedError: "archived repositories cannot be registered",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			mockEntityCreator := mock_entityservice.NewMockEntityCreator(ctrl)
			scenario.EntityCreator(mockEntityCreator)

			mockPropSvc := scenario.ServiceSetup(ctrl)
			mockEvents := mockevents.NewMockInterface(ctrl)
			mockEvents.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			svc := repositories.NewRepositoryService(nil, mockPropSvc, mockEvents, nil, mockEntityCreator)
			res, err := svc.CreateRepository(ctx, &provider, projectID, fetchByProps)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
				require.NotNil(t, res)
			} else {
				require.Nil(t, res)
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

// bundling delete by ID and name together due to the similarities of the tests
func TestRepositoryService_DeleteRepository(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name                 string
		DBSetup              dbMockBuilder
		ProviderManagerSetup func(provinfv1.Provider) pf.ProviderManagerMockBuilder
		ProviderSetup        providerMockBuilder
		ServiceSetup         propSvcMockBuilder
		DeleteType           DeleteCallType
		ExpectedError        string
	}{
		{
			Name:          "DeleteByName fails when repo cannot be retrieved",
			DeleteType:    ByName,
			DBSetup:       newDBMock(withFailedGetByName),
			ServiceSetup:  newPropSvcMock(),
			ProviderSetup: newProviderMock(),
			ProviderManagerSetup: func(_ provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock()
			},
			ExpectedError: errDefault.Error(),
		},
		{
			Name:          "DeleteByName fails when repo's entity cannot be retrieved",
			DeleteType:    ByName,
			DBSetup:       newDBMock(withSuccessfulGetByName),
			ServiceSetup:  newPropSvcMock(withFailedEntityWithProps),
			ProviderSetup: newProviderMock(),
			ProviderManagerSetup: func(_ provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock()
			},
			ExpectedError: errDefault.Error(),
		},
		{
			Name:         "DeleteByName fails when provider cannot be instantiated",
			DeleteType:   ByName,
			DBSetup:      newDBMock(withSuccessfulGetByName),
			ServiceSetup: newPropSvcMock(withSuccessfulEntityWithProps),
			ProviderManagerSetup: func(_ provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock(pf.WithFailedInstantiateFromID)
			},
			ProviderSetup: newProviderMock(),
			ExpectedError: "error instantiating provider",
		},
		{
			Name:         "DeleteByName still works when entity cannot be deregistered",
			DeleteType:   ByName,
			DBSetup:      newDBMock(withSuccessfulGetByName, withSuccessfulDelete),
			ServiceSetup: newPropSvcMock(withSuccessfulEntityWithProps),
			ProviderManagerSetup: func(p provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(p))
			},
			ProviderSetup: newProviderMock(withFailedDeregister),
		},
		{
			Name:         "DeleteByName by ID fails when repo cannot be deleted from DB",
			DeleteType:   ByName,
			DBSetup:      newDBMock(withSuccessfulGetByName, withFailedDelete),
			ServiceSetup: newPropSvcMock(withSuccessfulEntityWithProps),
			ProviderManagerSetup: func(p provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(p))
			},
			ProviderSetup: newProviderMock(withSuccessfulDeregister),
			ExpectedError: "error deleting entity from DB",
		},
		{
			Name:         "DeleteByName succeeds",
			DeleteType:   ByName,
			DBSetup:      newDBMock(withSuccessfulGetByName, withSuccessfulDelete),
			ServiceSetup: newPropSvcMock(withSuccessfulEntityWithProps),
			ProviderManagerSetup: func(p provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(p))
			},
			ProviderSetup: newProviderMock(withSuccessfulDeregister),
		},
		{
			Name:          "DeleteByID fails when repo entity cannot be retrieved",
			DeleteType:    ByID,
			DBSetup:       newDBMock(),
			ServiceSetup:  newPropSvcMock(withFailedEntityWithProps),
			ProviderSetup: newProviderMock(),
			ProviderManagerSetup: func(_ provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock()
			},
			ExpectedError: errDefault.Error(),
		},
		{
			Name:         "DeleteByID fails when provider cannot be instantiated",
			DeleteType:   ByID,
			DBSetup:      newDBMock(),
			ServiceSetup: newPropSvcMock(withSuccessfulEntityWithProps),
			ProviderManagerSetup: func(_ provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock(pf.WithFailedInstantiateFromID)
			},
			ProviderSetup: newProviderMock(),
			ExpectedError: "error instantiating provider",
		},
		{
			Name:         "DeleteByID works when entity cannot be deregistered",
			DeleteType:   ByID,
			DBSetup:      newDBMock(withSuccessfulDelete),
			ServiceSetup: newPropSvcMock(withSuccessfulEntityWithProps),
			ProviderManagerSetup: func(p provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(p))
			},
			ProviderSetup: newProviderMock(withFailedDeregister),
		},
		{
			Name:         "DeleteByID by ID fails when repo cannot be deleted from DB",
			DeleteType:   ByID,
			DBSetup:      newDBMock(withFailedDelete),
			ServiceSetup: newPropSvcMock(withSuccessfulEntityWithProps),
			ProviderManagerSetup: func(p provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(p))
			},
			ProviderSetup: newProviderMock(withSuccessfulDeregister),
			ExpectedError: "error deleting entity from DB",
		},
		{
			Name:         "DeleteByID succeeds",
			DeleteType:   ByID,
			DBSetup:      newDBMock(withSuccessfulDelete),
			ServiceSetup: newPropSvcMock(withSuccessfulEntityWithProps),
			ProviderManagerSetup: func(p provinfv1.Provider) pf.ProviderManagerMockBuilder {
				return pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(p))
			},
			ProviderSetup: newProviderMock(withSuccessfulDeregister),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			provm := scenario.ProviderSetup(ctrl)

			svc := createService(
				ctrl, scenario.DBSetup, scenario.ServiceSetup, scenario.ProviderManagerSetup(provm), false)
			var err error
			if scenario.DeleteType == ByName {
				err = svc.DeleteByName(ctx, dbRepo.RepoOwner, dbRepo.RepoName, projectID, providerName)
			} else {
				err = svc.DeleteByID(ctx, dbRepo.ID, projectID)
			}

			if scenario.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

func TestRepositoryService_GetRepositoryById(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		ServiceSetup  propSvcMockBuilder
		ShouldSucceed bool
	}{
		{
			Name:         "Get by ID fails when DB call fails",
			ServiceSetup: newPropSvcMock(withFailedEntityWithProps),
		},
		{
			Name: "Get by ID succeeds",
			ServiceSetup: newPropSvcMock(
				withSuccessfulEntityWithProps,
				withSuccessfulRetrieveAll,
				withSucessfulEntityToProto,
			),
			ShouldSucceed: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			svc := createService(ctrl, nil, scenario.ServiceSetup, nil, false)
			_, err := svc.GetRepositoryById(ctx, repoID, projectID)

			if scenario.ShouldSucceed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestRepositoryService_GetRepositoryByName(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		DBSetup       dbMockBuilder
		ServiceSetup  propSvcMockBuilder
		ShouldSucceed bool
	}{
		{
			Name:         "Get by name fails when DB call fails",
			DBSetup:      newDBMock(withFailedGetByName),
			ServiceSetup: newPropSvcMock(),
		},
		{
			Name:    "Get by name succeeds",
			DBSetup: newDBMock(withSuccessfulGetByName),
			ServiceSetup: newPropSvcMock(
				withSuccessfulEntityWithProps,
				withSuccessfulRetrieveAll,
				withSucessfulEntityToProto,
			),
			ShouldSucceed: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			svc := createService(ctrl, scenario.DBSetup, scenario.ServiceSetup, nil, false)
			_, err := svc.GetRepositoryByName(ctx, repoOwner, repoName, projectID, providerName)

			if scenario.ShouldSucceed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func createService(
	ctrl *gomock.Controller,
	dbSetup dbMockBuilder,
	serviceSetup propSvcMockBuilder,
	providerSetup pf.ProviderManagerMockBuilder,
	eventsFail bool,
) repositories.RepositoryService {
	var store db.Store
	if dbSetup != nil {
		store = dbSetup(ctrl)
	}
	var providerManager manager.ProviderManager
	if providerSetup != nil {
		providerManager = providerSetup(ctrl)
	}

	var eventErr error
	if eventsFail {
		eventErr = errDefault
	}
	events := mockevents.NewMockInterface(ctrl)
	events.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(eventErr).
		AnyTimes()

	mockPropSvc := serviceSetup(ctrl)

	// Create a mock entityCreator (we don't need to set expectations since CreateRepository
	// is called via the entityCreator now, but we keep the old test structure)
	mockEntityCreator := mock_entityservice.NewMockEntityCreator(ctrl)

	return repositories.NewRepositoryService(store, mockPropSvc, events, providerManager, mockEntityCreator)
}

const (
	repoOwner    = "acme-corp"
	repoName     = "api-gateway"
	providerName = "github"
)

type DeleteCallType int

const (
	ByID DeleteCallType = iota
	ByName
)

var (
	// ErrClientTest is a sample error used by the fixtures
	ErrClientTest = errors.New("oh no")
)

const (
	RepoOwner = "acme"
	RepoName  = "api-gateway"
	HookID    = int64(12345)
)

var (
	hookUUID   = uuid.New().String()
	repoID     = uuid.New()
	ghRepoID   = ptr.Ptr[int64](0xE1E10)
	projectID  = uuid.New()
	errDefault = errors.New("uh oh")
	// Test repository data - using a simple struct since db.Repository no longer exists
	dbRepo = struct {
		ID         uuid.UUID
		ProjectID  uuid.UUID
		RepoOwner  string
		RepoName   string
		ProviderID uuid.UUID
	}{
		ID:         repoID,
		ProjectID:  projectID,
		RepoOwner:  repoOwner,
		RepoName:   repoName,
		ProviderID: uuid.UUID{},
	}
	webhook = &gh.Hook{
		ID: ptr.Ptr[int64](HookID),
	}
	publicRepo   = newGithubRepo(false)
	fetchByProps = newFetchByGithubRepoProperties()
	publicProps  = newGithubRepoProperties(false)
	provider     = db.Provider{
		ID:         uuid.UUID{},
		Name:       providerName,
		Implements: []db.ProviderType{db.ProviderTypeGithub},
		Version:    provinfv1.V1,
	}
)

type (
	dbMock              = *mockdb.MockStore
	dbMockBuilder       = func(controller *gomock.Controller) dbMock
	propSvcMock         = *mock_propservice.MockPropertiesService
	propSvcMockBuilder  = func(controller *gomock.Controller) propSvcMock
	providerMock        = *mockgithub.MockGitHub
	providerMockBuilder = func(controller *gomock.Controller) providerMock
)

func newDBMock(opts ...func(dbMock)) dbMockBuilder {
	return func(ctrl *gomock.Controller) dbMock {
		mock := mockdb.NewMockStore(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func withFailedDelete(mock dbMock) {
	mock.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mock)
	mock.EXPECT().BeginTransaction().Return(nil, nil)
	mock.EXPECT().
		DeleteEntity(gomock.Any(), gomock.Any()).
		Return(errDefault)
	mock.EXPECT().Rollback(gomock.Any()).Return(nil)
}

func withSuccessfulDelete(mock dbMock) {
	mock.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mock)
	mock.EXPECT().BeginTransaction().Return(nil, nil)
	mock.EXPECT().
		DeleteEntity(gomock.Any(), gomock.Any()).
		Return(nil)
	mock.EXPECT().Commit(gomock.Any()).Return(nil)
	mock.EXPECT().Rollback(gomock.Any()).Return(nil)
}

func withFailedGetByName(mock dbMock) {
	// GetRepositoryByName now calls GetProviderByName + GetTypedEntitiesByPropertyV1
	mock.EXPECT().
		GetProviderByName(gomock.Any(), gomock.Any()).
		Return(provider, nil)
	mock.EXPECT().
		GetTypedEntitiesByPropertyV1(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]db.EntityInstance{}, errDefault)
}

func withSuccessfulGetByName(mock dbMock) {
	// GetRepositoryByName now calls GetProviderByName + GetTypedEntitiesByPropertyV1
	mock.EXPECT().
		GetProviderByName(gomock.Any(), gomock.Any()).
		Return(provider, nil)
	mock.EXPECT().
		GetTypedEntitiesByPropertyV1(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]db.EntityInstance{{ID: dbRepo.ID}}, nil)
}

func newGithubRepo(isPrivate bool) *gh.Repository {
	return &gh.Repository{
		ID:   ghRepoID,
		Name: ptr.Ptr(repoName),
		Owner: &gh.User{
			Login: ptr.Ptr(repoOwner),
		},
		Private:        ptr.Ptr(isPrivate),
		DeploymentsURL: ptr.Ptr("https://foo.com"),
		CloneURL:       ptr.Ptr("http://cloneurl.com"),
		Fork:           ptr.Ptr(false),
		DefaultBranch:  ptr.Ptr("main"),
	}
}

func newFetchByGithubRepoProperties() *properties.Properties {
	fetchByProps := map[string]any{
		properties.PropertyName:  fmt.Sprintf("%s/%s", repoOwner, repoName),
		ghprop.RepoPropertyName:  repoName,
		ghprop.RepoPropertyOwner: repoOwner,
	}

	props := properties.NewProperties(fetchByProps)
	return props
}

func newGithubRepoProperties(isPrivate bool) *properties.Properties {
	repoProps := map[string]any{
		properties.PropertyName:           fmt.Sprintf("%s/%s", repoOwner, repoName),
		properties.PropertyUpstreamID:     fmt.Sprintf("%d", *ghRepoID),
		properties.RepoPropertyIsPrivate:  isPrivate,
		properties.RepoPropertyIsArchived: false,
		properties.RepoPropertyIsFork:     false,
		ghprop.RepoPropertyId:             *ghRepoID,
		ghprop.RepoPropertyName:           repoName,
		ghprop.RepoPropertyOwner:          repoOwner,
		ghprop.RepoPropertyDeployURL:      "https://foo.com",
		ghprop.RepoPropertyCloneURL:       "http://cloneurl.com",
		ghprop.RepoPropertyDefaultBranch:  "main",
	}

	props := properties.NewProperties(repoProps)
	return props
}

func instantiatePBRepo(isPrivate bool) *pb.Repository {
	return &pb.Repository{
		Id:            ptr.Ptr(dbRepo.ID.String()),
		Name:          publicRepo.GetName(),
		Owner:         publicRepo.GetOwner().GetLogin(),
		RepoId:        publicRepo.GetID(),
		HookId:        webhook.GetID(),
		HookUrl:       webhook.GetURL(),
		DeployUrl:     publicRepo.GetDeploymentsURL(),
		CloneUrl:      publicRepo.GetCloneURL(),
		HookType:      webhook.GetType(),
		HookName:      webhook.GetName(),
		HookUuid:      hookUUID,
		IsPrivate:     isPrivate,
		IsFork:        publicRepo.GetFork(),
		DefaultBranch: publicRepo.GetDefaultBranch(),
	}
}

func newPropSvcMock(opts ...func(mock propSvcMock)) propSvcMockBuilder {
	return func(ctrl *gomock.Controller) propSvcMock {
		ms := mock_propservice.NewMockPropertiesService(ctrl)
		for _, opt := range opts {
			opt(ms)
		}
		return ms
	}
}

func withSuccessfulEntityWithProps(mock propSvcMock) {
	mock.EXPECT().
		EntityWithPropertiesByID(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(models.NewEntityWithPropertiesFromInstance(models.EntityInstance{
			ID:         dbRepo.ID,
			Type:       pb.Entity_ENTITY_REPOSITORIES,
			ProjectID:  projectID,
			ProviderID: dbRepo.ProviderID,
		}, publicProps), nil)
}

func withFailedEntityWithProps(mock propSvcMock) {
	mock.EXPECT().
		EntityWithPropertiesByID(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}

func withSucessfulEntityToProto(mock propSvcMock) {
	repo := instantiatePBRepo(false)
	mock.EXPECT().
		EntityWithPropertiesAsProto(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(repo, nil)
}

func withFailedEntityToProto(mock propSvcMock) {
	mock.EXPECT().
		EntityWithPropertiesAsProto(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}

func withSuccessfulRetrieveAll(mock propSvcMock) {
	mock.EXPECT().
		RetrieveAllPropertiesForEntity(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
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

func withSuccessfulDeregister(mock providerMock) {
	mock.EXPECT().
		DeregisterEntity(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
}

func withFailedDeregister(mock providerMock) {
	mock.EXPECT().
		DeregisterEntity(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errDefault)
}
