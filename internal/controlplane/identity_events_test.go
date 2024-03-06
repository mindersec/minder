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

	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/authz/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
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
	mockStore.EXPECT().Commit(gomock.Any())
	mockStore.EXPECT().Rollback(gomock.Any())

	mockStore.EXPECT().BeginTransaction().Return(&tx, nil)
	mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mockStore)
	mockStore.EXPECT().
		GetUserBySubject(gomock.Any(), "alreadyDeletedUserId").
		Return(db.User{}, sql.ErrNoRows)
	mockStore.EXPECT().Commit(gomock.Any())
	mockStore.EXPECT().Rollback(gomock.Any())

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
