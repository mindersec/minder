// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance cf.With the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repositories_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	gh "github.com/google/go-github/v63/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/entities/properties"
	mock_service "github.com/stacklok/minder/internal/entities/properties/service/mock"
	mockevents "github.com/stacklok/minder/internal/events/mock"
	mockgithub "github.com/stacklok/minder/internal/providers/github/mock"
	ghprop "github.com/stacklok/minder/internal/providers/github/properties"
	"github.com/stacklok/minder/internal/providers/manager"
	pf "github.com/stacklok/minder/internal/providers/manager/mock/fixtures"
	"github.com/stacklok/minder/internal/repositories"
	"github.com/stacklok/minder/internal/util/ptr"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func TestRepositoryService_CreateRepository(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name              string
		ProviderSetupFail bool
		ServiceSetup      propSvcMockBuilder
		DBSetup           dbMockBuilder
		EventsSetup       eventMockBuilder
		EventSendFails    bool
		ExpectedError     string
	}{
		{
			Name:              "CreateRepository fails when provider cannot be instantiated",
			ProviderSetupFail: true,
			ServiceSetup:      newPropSvcMock(),
			ExpectedError:     "error instantiating provider",
		},
		{
			Name:          "CreateRepository fails when repo properties cannot be found in GitHub",
			ServiceSetup:  newPropSvcMock(withFailingGet),
			ExpectedError: "error fetching properties for repository",
		},
		{
			Name:          "CreateRepository fails for private repo in project which disallows private repos",
			ServiceSetup:  newPropSvcMock(withSuccessfulPropFetch(privateProps)),
			DBSetup:       newDBMock(withPrivateReposDisabled),
			ExpectedError: "private repos cannot be registered in this project",
		},
		{
			Name:          "CreateRepository fails when webhook creation fails",
			ServiceSetup:  newPropSvcMock(withSuccessfulPropFetch(publicProps)),
			ExpectedError: "error creating webhook in repo",
		},
		{
			Name:          "CreateRepository fails when repo cannot be inserted into database",
			ServiceSetup:  newPropSvcMock(withSuccessfulPropFetch(publicProps)),
			DBSetup:       newDBMock(withFailedCreate),
			ExpectedError: "error creating repository",
		},
		{
			Name:          "CreateRepository fails when repo cannot be inserted into database (cleanup fails)",
			ServiceSetup:  newPropSvcMock(withSuccessfulPropFetch(publicProps)),
			DBSetup:       newDBMock(withFailedCreate),
			ExpectedError: "error creating repository",
		},
		{
			Name:         "CreateRepository succeeds",
			ServiceSetup: newPropSvcMock(withSuccessfulPropFetch(publicProps), withSuccessfulReplaceProps),
			DBSetup:      newDBMock(withSuccessfulCreate),
		},
		{
			Name:         "CreateRepository succeeds (private repos enabled)",
			ServiceSetup: newPropSvcMock(withSuccessfulPropFetch(privateProps), withSuccessfulReplaceProps),
			DBSetup:      newDBMock(withPrivateReposEnabled, withSuccessfulCreate),
		},
		{
			Name:           "CreateRepository succeeds (skips failed event send)",
			ServiceSetup:   newPropSvcMock(withSuccessfulPropFetch(publicProps), withSuccessfulReplaceProps),
			DBSetup:        newDBMock(withSuccessfulCreate),
			EventSendFails: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var opt func(mock pf.ProviderManagerMock)
			if !scenario.ProviderSetupFail {
				opt = pf.WithSuccessfulInstantiateFromID(mockgithub.NewMockGitHub(ctrl))
			} else {
				opt = pf.WithFailedInstantiateFromID
			}

			providerSetup := pf.NewProviderManagerMock(opt)

			svc := createService(ctrl, scenario.DBSetup, scenario.ServiceSetup, providerSetup, scenario.EventSendFails)
			res, err := svc.CreateRepository(ctx, &provider, projectID, repoOwner, repoName)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
				// cheat here a little...
				expectation := newExpectation(res.IsPrivate)
				require.Equal(t, expectation, res)
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
		Name          string
		DBSetup       dbMockBuilder
		ProviderSetup pf.ProviderManagerMockBuilder
		DeleteType    DeleteCallType
		ExpectedError string
	}{
		{
			Name:          "DeleteByName fails when repo cannot be retrieved",
			DeleteType:    ByName,
			DBSetup:       newDBMock(withFailedGetByName),
			ExpectedError: "error retrieving repository",
		},
		{
			Name:          "DeleteByName fails when provider cannot be instantiated",
			DeleteType:    ByName,
			DBSetup:       newDBMock(withSuccessfulGetByName),
			ProviderSetup: pf.NewProviderManagerMock(pf.WithFailedInstantiateFromID),
			ExpectedError: "error while instantiating provider",
		},
		{
			Name:          "DeleteByName fails when webhook cannot be deleted",
			DeleteType:    ByName,
			DBSetup:       newDBMock(withSuccessfulGetByName),
			ProviderSetup: pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(ghProvider)),
			ExpectedError: "error deleting webhook",
		},
		{
			Name:          "DeleteByName by ID fails when repo cannot be deleted from DB",
			DeleteType:    ByName,
			DBSetup:       newDBMock(withSuccessfulGetByName, withFailedDelete),
			ProviderSetup: pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(ghProvider)),
			ExpectedError: "error deleting repository from DB",
		},
		{
			Name:          "DeleteByName succeeds",
			DeleteType:    ByName,
			DBSetup:       newDBMock(withSuccessfulGetByName, withSuccessfulDelete),
			ProviderSetup: pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(ghProvider)),
		},
		{
			Name:          "DeleteByID fails when repo cannot be retrieved",
			DeleteType:    ByID,
			DBSetup:       newDBMock(withFailedGetById),
			ExpectedError: "error retrieving repository",
		},
		{
			Name:          "DeleteByID fails when provider cannot be instantiated",
			DeleteType:    ByID,
			DBSetup:       newDBMock(withSuccessfulGetById),
			ProviderSetup: pf.NewProviderManagerMock(pf.WithFailedInstantiateFromID),
			ExpectedError: "error while instantiating provider",
		},
		{
			Name:          "DeleteByID fails when webhook cannot be deleted",
			DeleteType:    ByID,
			DBSetup:       newDBMock(withSuccessfulGetById),
			ProviderSetup: pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(ghProvider)),
			ExpectedError: "error deleting webhook",
		},
		{
			Name:          "DeleteByID by ID fails when repo cannot be deleted from DB",
			DeleteType:    ByID,
			DBSetup:       newDBMock(withSuccessfulGetById, withFailedDelete),
			ProviderSetup: pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(ghProvider)),
			ExpectedError: "error deleting repository from DB",
		},
		{
			Name:          "DeleteByID succeeds",
			DeleteType:    ByID,
			DBSetup:       newDBMock(withSuccessfulGetById, withSuccessfulDelete),
			ProviderSetup: pf.NewProviderManagerMock(pf.WithSuccessfulInstantiateFromID(ghProvider)),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			svc := createService(ctrl, scenario.DBSetup, newPropSvcMock(), scenario.ProviderSetup, false)
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
		DBSetup       dbMockBuilder
		DeleteType    DeleteCallType
		ShouldSucceed bool
	}{
		{
			Name:    "Get by ID fails when DB call fails",
			DBSetup: newDBMock(withFailedGetById),
		},
		{
			Name:          "Get by ID succeeds",
			DBSetup:       newDBMock(withSuccessfulGetById),
			ShouldSucceed: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			// For the purposes of this test, we do not need to attach any
			// mock behaviour to the client. We can leave it as a nil pointer.

			svc := createService(ctrl, scenario.DBSetup, newPropSvcMock(), nil, false)
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
		DeleteType    DeleteCallType
		ShouldSucceed bool
	}{
		{
			Name:    "Get by name fails when DB call fails",
			DBSetup: newDBMock(withFailedGetByName),
		},
		{
			Name:          "Get by name succeeds",
			DBSetup:       newDBMock(withSuccessfulGetByName),
			ShouldSucceed: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			// For the purposes of this test, we do not need to attach any
			// mock behaviour to the client. We can leave it as a nil pointer.

			svc := createService(ctrl, scenario.DBSetup, newPropSvcMock(), nil, false)
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

	return repositories.NewRepositoryService(store, mockPropSvc, events, providerManager)
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
	hookUUID   = uuid.New().String()
	repoID     = uuid.New()
	ghRepoID   = ptr.Ptr[int64](0xE1E10)
	projectID  = uuid.New()
	errDefault = errors.New("uh oh")
	dbRepo     = db.Repository{
		ID:        repoID,
		ProjectID: projectID,
		RepoOwner: repoOwner,
		RepoName:  repoName,
		WebhookID: sql.NullInt64{
			Valid: true,
			Int64: cf.HookID,
		},
	}
	webhook = &gh.Hook{
		ID: ptr.Ptr[int64](cf.HookID),
	}
	webhookProps = newWebhookProperties(cf.HookID, hookUUID)
	publicRepo   = newGithubRepo(false)
	publicProps  = newGithubRepoProperties(false)
	privateProps = newGithubRepoProperties(true)
	provider     = db.Provider{
		ID:         uuid.UUID{},
		Name:       providerName,
		Implements: []db.ProviderType{db.ProviderTypeGithub},
		Version:    provinfv1.V1,
	}
	ghProvider = mockgithub.NewMockGitHub(nil)
)

type (
	dbMock             = *mockdb.MockStore
	dbMockBuilder      = func(controller *gomock.Controller) dbMock
	propSvcMock        = *mock_service.MockPropertiesService
	propSvcMockBuilder = func(controller *gomock.Controller) propSvcMock
	eventMock          = *mockevents.MockInterface
	eventMockBuilder   = func(controller *gomock.Controller) eventMock
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
		DeleteRepository(gomock.Any(), gomock.Eq(repoID)).
		Return(errDefault)
	mock.EXPECT().Rollback(gomock.Any()).Return(nil)
}

func withSuccessfulDelete(mock dbMock) {
	mock.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mock)
	mock.EXPECT().BeginTransaction().Return(nil, nil)
	mock.EXPECT().
		DeleteRepository(gomock.Any(), gomock.Eq(repoID)).
		Return(nil)
	mock.EXPECT().
		DeleteEntity(gomock.Any(), gomock.Any()).
		Return(nil)
	mock.EXPECT().Commit(gomock.Any()).Return(nil)
	mock.EXPECT().Rollback(gomock.Any()).Return(nil)
}

func withFailedGetById(mock dbMock) {
	mock.EXPECT().
		GetRepositoryByIDAndProject(gomock.Any(), gomock.Any()).
		Return(db.Repository{}, errDefault)
}

func withSuccessfulGetById(mock dbMock) {
	mock.EXPECT().
		GetRepositoryByIDAndProject(gomock.Any(), gomock.Any()).
		Return(dbRepo, nil)
}

func withFailedGetByName(mock dbMock) {
	mock.EXPECT().
		GetRepositoryByRepoName(gomock.Any(), gomock.Any()).
		Return(db.Repository{}, errDefault)
}

func withSuccessfulGetByName(mock dbMock) {
	mock.EXPECT().
		GetRepositoryByRepoName(gomock.Any(), gomock.Any()).
		Return(dbRepo, nil)
}

func withFailedCreate(mock dbMock) {
	mock.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mock)
	mock.EXPECT().BeginTransaction().Return(nil, nil)
	mock.EXPECT().
		CreateRepository(gomock.Any(), gomock.Any()).
		Return(db.Repository{}, errDefault)
	mock.EXPECT().Rollback(gomock.Any()).Return(nil)
}

func withSuccessfulCreate(mock dbMock) {
	mock.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mock)
	mock.EXPECT().BeginTransaction().Return(nil, nil)
	mock.EXPECT().
		CreateRepository(gomock.Any(), gomock.Any()).
		Return(dbRepo, nil)
	mock.EXPECT().
		CreateEntityWithID(gomock.Any(), gomock.Any()).
		Return(db.EntityInstance{}, nil)
	mock.EXPECT().Commit(gomock.Any()).Return(nil)
	mock.EXPECT().Rollback(gomock.Any()).Return(nil)
}

func withPrivateReposEnabled(mock dbMock) {
	mock.EXPECT().
		GetFeatureInProject(gomock.Any(), gomock.Any()).
		Return(json.RawMessage{}, nil)
}

func withPrivateReposDisabled(mock dbMock) {
	mock.EXPECT().
		GetFeatureInProject(gomock.Any(), gomock.Any()).
		Return(json.RawMessage{}, sql.ErrNoRows)
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

func newWebhookProperties(hookID int64, hookUUID string) *properties.Properties {
	webhookProps := map[string]any{
		ghprop.RepoPropertyHookId:   hookID,
		ghprop.RepoPropertyHookUiid: hookUUID,
	}

	props, err := properties.NewProperties(webhookProps)
	if err != nil {
		panic(err)
	}
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

	props, err := properties.NewProperties(repoProps)
	if err != nil {
		panic(err)
	}
	return props
}

func newExpectation(isPrivate bool) *pb.Repository {
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
		ms := mock_service.NewMockPropertiesService(ctrl)
		for _, opt := range opts {
			opt(ms)
		}
		return ms
	}
}

func withSuccessfulReplaceProps(mock propSvcMock) {
	mock.EXPECT().
		ReplaceAllProperties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
}

func withFailingGet(mock propSvcMock) {
	mock.EXPECT().
		RetrieveAllProperties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errDefault)
}

func withSuccessfulPropFetch(prop *properties.Properties) func(svcMock propSvcMock) {
	return func(mock propSvcMock) {
		mock.EXPECT().
			RetrieveAllProperties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(prop, nil)
	}
}
