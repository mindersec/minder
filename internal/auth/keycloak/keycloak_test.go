// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package keycloak provides an implementation of the Keycloak IdentityProvider.
package keycloak

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/keycloak/client"
	"github.com/mindersec/minder/internal/util/ptr"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

func TestKeyCloak_Resolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		id           string
		federatedIdP string
		wantError    string
		human        string
		userid       string
	}{{
		name:   "Keycloak id",
		id:     "1a311ff9-4478-4866-a14a-b1eeacf0c0c0",
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
		wantError: "user not found in identity store",
	}, {
		name:         "GitHub id - federated",
		id:           "123456",
		federatedIdP: "github",
		human:        "user",
		userid:       "1a311ff9-4478-4866-a14a-b1eeacf0c0c0",
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
		Realm:     "stacklok",
	})
	if err != nil {
		t.Fatalf("failed to create keycloak: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			var id *auth.Identity
			var err error
			if tt.federatedIdP != "" {
				id, err = kc.ResolveFederated(ctx, tt.federatedIdP, tt.id)
			} else {
				id, err = kc.Resolve(ctx, tt.id)
			}

			if tt.wantError != "" {
				assert.ErrorIs(t, err, auth.ErrNotFound)
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

func TestKeyCloak_ResolveLogsMisses(t *testing.T) {
	t.Parallel()

	user := client.UserRepresentation{
		Attributes: &map[string][]string{"gh_id": {"123456"}},
		Id:         ptr.Ptr("1a311ff9-4478-4866-a14a-b1eeacf0c0c0"),
		Username:   ptr.Ptr("user"),
	}

	tests := []struct {
		name string
		// adminStatus, if set, is the status the admin API answers every call with.
		adminStatus int
		federated   bool
		id          string
		// wantLog is the expected log line, or nil if nothing should be logged.
		wantLog map[string]any
	}{{
		name:    "found by username logs nothing",
		id:      "user",
		wantLog: nil,
	}, {
		name: "unknown user logs at debug",
		id:   "unknown",
		wantLog: map[string]any{
			"level":       "debug",
			"by_id":       map[string]any{"status": float64(404)},
			"by_username": map[string]any{"status": float64(200), "matches": float64(0)},
		},
	}, {
		name:        "admin API forbidden",
		adminStatus: http.StatusForbidden,
		id:          "user",
		wantLog: map[string]any{
			"level":       "warn",
			"by_id":       map[string]any{"status": float64(403)},
			"by_username": map[string]any{"status": float64(403)},
		},
	}, {
		name:        "admin API not routed",
		adminStatus: http.StatusNotFound,
		id:          "user",
		wantLog: map[string]any{
			"level":       "warn",
			"by_id":       map[string]any{"status": float64(404)},
			"by_username": map[string]any{"status": float64(404)},
		},
	}, {
		name:        "federated lookup forbidden",
		adminStatus: http.StatusForbidden,
		federated:   true,
		id:          "123456",
		wantLog: map[string]any{
			"level":        "warn",
			"by_github_id": map[string]any{"status": float64(403)},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeKeycloak{
				users:       map[string]client.UserRepresentation{*user.Id: user},
				adminStatus: tt.adminStatus,
			}
			fakeServ := fake.Start(t)
			t.Cleanup(fakeServ.Close)
			kcClient, err := client.NewClientWithResponses(fakeServ.URL)
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}
			kc := &KeyCloak{realm: "stacklok", kcClient: kcClient}

			var buf bytes.Buffer
			ctx := zerolog.New(&buf).WithContext(context.Background())
			if tt.federated {
				_, err = kc.ResolveFederated(ctx, "github", tt.id)
			} else {
				_, err = kc.Resolve(ctx, tt.id)
			}

			if tt.wantLog == nil {
				assert.NoError(t, err)
				assert.Empty(t, buf.String())
				return
			}
			assert.ErrorIs(t, err, auth.ErrNotFound)
			var got map[string]any
			if uerr := json.Unmarshal(buf.Bytes(), &got); uerr != nil {
				t.Fatalf("expected one JSON log line, got %q: %v", buf.String(), uerr)
			}
			for key, want := range tt.wantLog {
				assert.Equal(t, want, got[key], key)
			}
		})
	}
}

type fakeKeycloak struct {
	users map[string]client.UserRepresentation
	// adminStatus, if set, makes the admin API answer every call with it.
	adminStatus int
}

func (f *fakeKeycloak) Start(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/admin/realms/stacklok/users/{userid}", f.GetUser)
	mux.HandleFunc("/admin/realms/stacklok/users", f.GetUserByQuery)
	mux.HandleFunc("/realms/stacklok/protocol/openid-connect/token", f.GetToken)
	mux.HandleFunc("/realms/stacklok/.well-known/openid-configuration", f.GetOIDCConfig)
	mux.HandleFunc("/", LogMissing(t))

	return httptest.NewServer(mux)
}

func (*fakeKeycloak) GetToken(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"access_token":"1234","expires_in":300,"token_type":"Bearer"}`)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (*fakeKeycloak) GetOIDCConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{
		"issuer": "http://` + r.Host + `/realms/stacklok",
		"jwks_uri": "http://` + r.Host + `/realms/stacklok/protocol/openid-connect/certs",
		"token_endpoint": "http://` + r.Host + `/realms/stacklok/protocol/openid-connect/token"
	}`)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (f *fakeKeycloak) GetUser(w http.ResponseWriter, r *http.Request) {
	if f.adminStatus != 0 {
		http.Error(w, http.StatusText(f.adminStatus), f.adminStatus)
		return
	}
	user, ok := f.users[r.PathValue("userid")]
	if !ok {
		// Keycloak answers 404 for an unknown user ID.
		http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w)
	if err := e.Encode(user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (f *fakeKeycloak) GetUserByQuery(w http.ResponseWriter, r *http.Request) {
	if f.adminStatus != 0 {
		http.Error(w, http.StatusText(f.adminStatus), f.adminStatus)
		return
	}
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
	// Keycloak answers a search with no matches with an empty list.
	if _, err := w.Write([]byte("[]")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func LogMissing(t *testing.T) func(w http.ResponseWriter, r *http.Request) {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		t.Logf("missing handler for %s\n", r.URL.Path)
		http.Error(w, "missing handler", http.StatusNotFound)
	}
}
