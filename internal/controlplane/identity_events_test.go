// Copyright 2023 Stacklok, Inc
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

package controlplane

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/authz/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/db/embedded"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestHandleEvents(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/stacklok/protocol/openid-connect/token":
			tokenHandler(t, w)
		case "/admin/realms/stacklok/events":
			eventHandler(t, w)
		default:
			t.Fatalf("Unexpected call to mock server endpoint %s", r.URL.Path)
		}
	}))
	defer server.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	tx := sql.Tx{}
	mockStore.EXPECT().BeginTransaction().Return(&tx, nil)
	mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mockStore)
	mockStore.EXPECT().
		GetUserBySubject(gomock.Any(), "existingUserId").
		Return(db.User{
			IdentitySubject: "existingUserId",
		}, nil)
	mockStore.EXPECT().
		DeleteUser(gomock.Any(), gomock.Any()).
		Return(nil)
	mockStore.EXPECT().Commit(gomock.Any())

	mockStore.EXPECT().BeginTransaction().Return(&tx, nil)
	mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mockStore)
	mockStore.EXPECT().
		GetUserBySubject(gomock.Any(), "alreadyDeletedUserId").
		Return(db.User{}, sql.ErrNoRows)
	mockStore.EXPECT().Commit(gomock.Any())

	c := serverconfig.Config{
		Identity: serverconfig.IdentityConfigWrapper{
			Server: serverconfig.IdentityConfig{
				IssuerUrl:    server.URL,
				ClientId:     "client-id",
				ClientSecret: "client-secret",
			},
		},
	}
	HandleEvents(context.Background(), mockStore, &mock.NoopClient{Authorized: true}, &c)
}

func TestDeleteUserOneProject(t *testing.T) {
	t.Parallel()

	store, td, err := embedded.GetFakeStore()
	require.NoError(t, err)

	t.Cleanup(td)

	ctx, tmout := context.WithTimeout(context.Background(), 5*time.Second)
	defer tmout()

	t.Log("Creating test project")
	p1, err := store.CreateProject(context.Background(), db.CreateProjectParams{
		Name:     t.Name(),
		Metadata: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	t.Log("Creating test user")
	u1, err := store.CreateUser(context.Background(), t.Name())
	require.NoError(t, err)

	pID := p1.ID.String()
	authzClient := mock.SimpleClient{
		Allowed: []uuid.UUID{p1.ID},
		Assignments: map[uuid.UUID][]*minderv1.RoleAssignment{
			p1.ID: {
				{
					Subject: u1.IdentitySubject,
					Role:    authz.AuthzRoleAdmin.String(),
					Project: &pID,
				},
			},
		},
	}

	t.Log("Deleting user")
	err = DeleteUser(ctx, store, &authzClient, u1.IdentitySubject)
	assert.NoError(t, err, "DeleteUser failed")

	t.Log("Checking if user is removed from project")
	u, err := store.GetUserBySubject(context.Background(), u1.IdentitySubject)
	assert.Error(t, err, "User not deleted")
	assert.ErrorIs(t, err, sql.ErrNoRows, "User not deleted")
	t.Logf("User: %+v", u)

	t.Log("Checking if user is removed from project")
	assignments, err := authzClient.AssignmentsToProject(ctx, p1.ID)
	assert.NoError(t, err, "AssignmentsToProject failed")
	assert.Empty(t, assignments, "User not removed from project")

	t.Log("Checking if project is removed")
	_, err = store.GetProjectByID(context.Background(), p1.ID)
	assert.ErrorIs(t, err, sql.ErrNoRows, "Project not deleted")
}

func tokenHandler(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	data := oauth2.Token{
		AccessToken: "some-token",
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		t.Fatal(err)
	}
}

func eventHandler(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	data := []AccountEvent{
		{
			Time:     1697030342912,
			Type:     "DELETE_ACCOUNT",
			RealmId:  "realmId",
			ClientId: "clientId",
			UserId:   "existingUserId",
		},
		{
			Time:     1697030342844,
			Type:     "DELETE_ACCOUNT",
			RealmId:  "realmId",
			ClientId: "clientId",
			UserId:   "alreadyDeletedUserId",
		},
	}
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		t.Fatal(err)
	}
}
