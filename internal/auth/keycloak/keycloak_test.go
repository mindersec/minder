// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package keycloak provides an implementation of the Keycloak IdentityProvider.
package keycloak

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"

	"github.com/mindersec/minder/internal/auth/keycloak/client"
	serverconfig "github.com/mindersec/minder/internal/config/server"
	"github.com/mindersec/minder/pkg/util/ptr"
)

func TestKeyCloak_Resolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		id        string
		wantError string
		human     string
		userid    string
	}{{
		name:   "Keycloak id",
		id:     "1a311ff9-4478-4866-a14a-b1eeacf0c0c0",
		human:  "user",
		userid: "1a311ff9-4478-4866-a14a-b1eeacf0c0c0",
	}, {
		name:   "GitHub id",
		id:     "123456",
		human:  "user",
		userid: "1a311ff9-4478-4866-a14a-b1eeacf0c0c0",
	}, {
		name:   "GitHub login",
		id:     "user",
		human:  "user",
		userid: "1a311ff9-4478-4866-a14a-b1eeacf0c0c0",
	}, {
		name:      "Unknown id",
		id:        "unknown",
		wantError: "unable to resolve user: user not found in identity store",
	}}

	fakeKeycloak := &fakeKeycloak{
		users: map[string]client.UserRepresentation{
			"1a311ff9-4478-4866-a14a-b1eeacf0c0c0": {
				Attributes: &map[string][]string{
					"gh_login": {"user"},
					"gh_id":    {"123456"},
				},
				Id:       ptr.Ptr("1a311ff9-4478-4866-a14a-b1eeacf0c0c0"),
				Username: ptr.Ptr("user"),
			},
		},
	}
	fakeServ := fakeKeycloak.Start(t)
	t.Cleanup(fakeServ.Close)

	kc, err := NewKeyCloak("", serverconfig.IdentityConfig{
		IssuerUrl: fakeServ.URL,
	})
	if err != nil {
		t.Fatalf("failed to create keycloak: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			id, err := kc.Resolve(ctx, tt.id)
			if tt.wantError != "" {
				assert.Equal(t, tt.wantError, err.Error())
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.human, id.Human())
			assert.Equal(t, tt.userid, id.String())

			userJWT := jwt.New()
			assert.NoError(t, userJWT.Set("sub", tt.userid))
			assert.NoError(t, userJWT.Set("iss", fakeServ.URL+"/realms/stacklok"))
			assert.NoError(t, userJWT.Set("preferred_username", tt.human))
			id, err = kc.Validate(ctx, userJWT)
			assert.NoError(t, err)
			assert.Equal(t, tt.human, id.Human())
			assert.Equal(t, tt.userid, id.String())
		})
	}
}

type fakeKeycloak struct {
	users map[string]client.UserRepresentation
}

func (f *fakeKeycloak) Start(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/admin/realms/stacklok/users/{userid}", f.GetUser)
	mux.HandleFunc("/admin/realms/stacklok/users", f.GetUserByQuery)
	mux.HandleFunc("/realms/stacklok/protocol/openid-connect/token", f.GetToken)
	mux.HandleFunc("/", LogMissing(t))

	return httptest.NewServer(mux)
}

func (_ *fakeKeycloak) GetToken(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"access_token":"1234","expires_in":300,"token_type":"Bearer"}`)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (f *fakeKeycloak) GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	user := r.PathValue("userid")
	e := json.NewEncoder(w)
	if err := e.Encode(f.users[user]); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (f *fakeKeycloak) GetUserByQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	matcher := func(u client.UserRepresentation) bool {
		if r.URL.Query().Get("username") != "" {
			return *u.Username == r.URL.Query().Get("username")
		}
		if r.URL.Query().Get("q") != "" {
			parsed := strings.Split(r.URL.Query().Get("q"), ":")
			return (*u.Attributes)[parsed[0]][0] == parsed[1]
		}
		return false
	}
	for _, u := range f.users {
		if matcher(u) {
			results := []client.UserRepresentation{u}
			e := json.NewEncoder(w)
			if err := e.Encode(results); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	http.Error(w, "Not Found", http.StatusInternalServerError)
}

func LogMissing(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		t.Logf("missing handler for %s\n", r.URL.Path)
		http.Error(w, "missing handler", http.StatusNotFound)
	}
}
