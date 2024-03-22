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

package github_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	gh "github.com/google/go-github/v56/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/db"
	mockevents "github.com/stacklok/minder/internal/events/mock"
	"github.com/stacklok/minder/internal/repositories/github"
	"github.com/stacklok/minder/internal/repositories/github/clients"
	cf "github.com/stacklok/minder/internal/repositories/github/fixtures"
	"github.com/stacklok/minder/internal/repositories/github/webhooks"
	mockghhook "github.com/stacklok/minder/internal/repositories/github/webhooks/mock"
	"github.com/stacklok/minder/internal/util/ptr"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func TestRepositoryService_CreateRepository(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name           string
		ClientSetup    cf.ClientMockBuilder
		DBSetup        dbMockBuilder
		WebhookSetup   whMockBuilder
		EventsSetup    eventMockBuilder
		EventSendFails bool
		ExpectedError  string
	}{
		{
			Name:          "CreateRepository fails when repo cannot be found in GitHub",
			ClientSetup:   cf.NewClientMock(cf.WithFailedGet),
			ExpectedError: "error retrieving repo from github",
		},
		{
			Name:          "CreateRepository fails for private repo in project which disallows private repos",
			ClientSetup:   cf.NewClientMock(cf.WithSuccessfulGet(privateRepo)),
			DBSetup:       newDBMock(withPrivateReposDisabled),
			ExpectedError: "private repos cannot be registered in this project",
		},
		{
			Name:          "CreateRepository fails when webhook creation fails",
			ClientSetup:   cf.NewClientMock(cf.WithSuccessfulGet(publicRepo)),
			WebhookSetup:  newWebhookMock(withFailedWebhookCreate),
			ExpectedError: "error creating webhook in repo",
		},
		{
			Name:          "CreateRepository fails when repo cannot be inserted into database",
			ClientSetup:   cf.NewClientMock(cf.WithSuccessfulGet(publicRepo)),
			DBSetup:       newDBMock(withFailedCreate),
			WebhookSetup:  newWebhookMock(withSuccessfulWebhookCreate, withSuccessfulWebhookDelete),
			ExpectedError: "error creating repository",
		},
		{
			Name:          "CreateRepository fails when repo cannot be inserted into database (cleanup fails)",
			ClientSetup:   cf.NewClientMock(cf.WithSuccessfulGet(publicRepo)),
			DBSetup:       newDBMock(withFailedCreate),
			WebhookSetup:  newWebhookMock(withSuccessfulWebhookCreate, withFailedWebhookDelete),
			ExpectedError: "error creating repository",
		},
		{
			Name:         "CreateRepository succeeds",
			ClientSetup:  cf.NewClientMock(cf.WithSuccessfulGet(publicRepo)),
			DBSetup:      newDBMock(withSuccessfulCreate),
			WebhookSetup: newWebhookMock(withSuccessfulWebhookCreate),
		},
		{
			Name:         "CreateRepository succeeds (private repos enabled)",
			ClientSetup:  cf.NewClientMock(cf.WithSuccessfulGet(privateRepo)),
			DBSetup:      newDBMock(withPrivateReposEnabled, withSuccessfulCreate),
			WebhookSetup: newWebhookMock(withSuccessfulWebhookCreate),
		},
		{
			Name:           "CreateRepository succeeds (skips failed event send)",
			ClientSetup:    cf.NewClientMock(cf.WithSuccessfulGet(publicRepo)),
			DBSetup:        newDBMock(withSuccessfulCreate),
			WebhookSetup:   newWebhookMock(withSuccessfulWebhookCreate),
			EventSendFails: true,
		},
	}

	for i := range scenarios {
		scenario := scenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var ghClient clients.GitHubRepoClient
			if scenario.ClientSetup != nil {
				ghClient = scenario.ClientSetup(ctrl)
			}

			svc := createService(ctrl, scenario.WebhookSetup, scenario.DBSetup, scenario.EventSendFails)
			res, err := svc.CreateRepository(ctx, ghClient, &provider, projectID, repoOwner, repoName)
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

func TestRepositoryService_DeleteRepository(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		DBSetup       dbMockBuilder
		WebhookSetup  whMockBuilder
		DeleteType    DeleteCallType
		ShouldSucceed bool
	}{
		{
			Name:       "Delete by ID fails when repo does not exist in DB",
			DBSetup:    newDBMock(withNotFoundByID),
			DeleteType: ByID,
		},
		{
			Name:       "Delete by Name fails when repo does not exist in DB",
			DBSetup:    newDBMock(withNotFoundByName),
			DeleteType: ByName,
		},
		{
			Name:         "Delete by ID fails when webhook cannot be deleted",
			DBSetup:      newDBMock(withRepoByID),
			WebhookSetup: newWebhookMock(withFailedWebhookDelete),
			DeleteType:   ByID,
		},
		{
			Name:         "Delete by Name fails when webhook cannot be deleted",
			DBSetup:      newDBMock(withRepoByName),
			WebhookSetup: newWebhookMock(withFailedWebhookDelete),
			DeleteType:   ByName,
		},
		{
			Name:         "Delete by ID fails when repo cannot be deleted from DB",
			DBSetup:      newDBMock(withRepoByID, withFailedDelete),
			WebhookSetup: newWebhookMock(withSuccessfulWebhookDelete),
			DeleteType:   ByID,
		},
		{
			Name:         "Delete by Name fails when repo cannot be deleted from DB",
			DBSetup:      newDBMock(withRepoByName, withFailedDelete),
			WebhookSetup: newWebhookMock(withSuccessfulWebhookDelete),
			DeleteType:   ByName,
		},
		{
			Name:          "Delete by ID succeeds",
			DBSetup:       newDBMock(withRepoByID, withSuccessfulDelete),
			WebhookSetup:  newWebhookMock(withSuccessfulWebhookDelete),
			DeleteType:    ByID,
			ShouldSucceed: true,
		},
		{
			Name:          "Delete by Name succeeds",
			DBSetup:       newDBMock(withRepoByName, withSuccessfulDelete),
			WebhookSetup:  newWebhookMock(withSuccessfulWebhookDelete),
			DeleteType:    ByName,
			ShouldSucceed: true,
		},
	}

	for i := range scenarios {
		scenario := scenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			// For the purposes of this test, we do not need to attach any
			// mock behaviour to the client. We can leave it as a nil pointer.
			var ghClient clients.GitHubRepoClient

			svc := createService(ctrl, scenario.WebhookSetup, scenario.DBSetup, false)
			var err error
			switch scenario.DeleteType {
			case ByID:
				err = svc.DeleteRepositoryByID(ctx, ghClient, projectID, repoID)
			case ByName:
				err = svc.DeleteRepositoryByName(ctx, ghClient, projectID, providerName, repoOwner, repoName)
			default:
				t.Fatalf("Unknown deletion type: %d", scenario.DeleteType)
			}

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
	whSetup whMockBuilder,
	dbSetup dbMockBuilder,
	eventsFail bool,
) github.RepositoryService {
	var store db.Store
	if dbSetup != nil {
		store = dbSetup(ctrl)
	}
	var whManager webhooks.WebhookManager
	if whSetup != nil {
		whManager = whSetup(ctrl)
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

	return github.NewRepositoryService(whManager, store, events)
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
	publicRepo  = newGithubRepo(false)
	privateRepo = newGithubRepo(true)
	provider    = db.Provider{
		ID:         uuid.UUID{},
		Name:       providerName,
		Implements: []db.ProviderType{db.ProviderTypeGithub},
		Version:    provinfv1.V1,
	}
)

type (
	dbMock           = *mockdb.MockStore
	dbMockBuilder    = func(controller *gomock.Controller) dbMock
	whMock           = *mockghhook.MockWebhookManager
	whMockBuilder    = func(controller *gomock.Controller) whMock
	eventMock        = *mockevents.MockInterface
	eventMockBuilder = func(controller *gomock.Controller) eventMock
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

func newWebhookMock(opts ...func(mock whMock)) whMockBuilder {
	return func(ctrl *gomock.Controller) whMock {
		mock := mockghhook.NewMockWebhookManager(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func withSuccessfulWebhookCreate(mock whMock) {
	mock.EXPECT().
		CreateWebhook(gomock.Any(), gomock.Any(), repoOwner, repoName).
		Return(hookUUID, webhook, nil)
}

func withFailedWebhookCreate(mock whMock) {
	mock.EXPECT().
		CreateWebhook(gomock.Any(), gomock.Any(), repoOwner, repoName).
		Return("", nil, errDefault)
}

func withSuccessfulWebhookDelete(mock whMock) {
	mock.EXPECT().
		DeleteWebhook(gomock.Any(), gomock.Any(), repoOwner, repoName, cf.HookID).
		Return(nil)
}

func withFailedWebhookDelete(mock whMock) {
	mock.EXPECT().
		DeleteWebhook(gomock.Any(), gomock.Any(), repoOwner, repoName, cf.HookID).
		Return(errDefault)
}

func withNotFoundByID(mock dbMock) {
	mock.EXPECT().
		GetRepositoryByIDAndProject(gomock.Any(), gomock.Any()).
		Return(db.Repository{}, errDefault)
}

func withNotFoundByName(mock dbMock) {
	mock.EXPECT().
		// TODO: do we want more strict assertions here?
		GetRepositoryByRepoName(gomock.Any(), gomock.Any()).
		Return(db.Repository{}, errDefault)
}

func withRepoByID(mock dbMock) {
	mock.EXPECT().
		GetRepositoryByIDAndProject(gomock.Any(), gomock.Any()).
		Return(dbRepo, nil)
}

func withRepoByName(mock dbMock) {
	mock.EXPECT().
		GetRepositoryByRepoName(gomock.Any(), gomock.Any()).
		Return(dbRepo, nil)
}

func withFailedDelete(mock dbMock) {
	mock.EXPECT().
		DeleteRepository(gomock.Any(), gomock.Eq(repoID)).
		Return(errDefault)
}

func withSuccessfulDelete(mock dbMock) {
	mock.EXPECT().
		DeleteRepository(gomock.Any(), gomock.Eq(repoID)).
		Return(nil)
}

func withFailedCreate(mock dbMock) {
	mock.EXPECT().
		CreateRepository(gomock.Any(), gomock.Any()).
		Return(db.Repository{}, errDefault)
}

func withSuccessfulCreate(mock dbMock) {
	mock.EXPECT().
		CreateRepository(gomock.Any(), gomock.Any()).
		Return(dbRepo, nil)
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
