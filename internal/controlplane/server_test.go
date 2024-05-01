//
// Copyright 2023 Stacklok, Inc.
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

package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	mockdb "github.com/stacklok/minder/database/mock"
	mockjwt "github.com/stacklok/minder/internal/auth/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/ratecache"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

var httpServer *httptest.Server

func init() {
	// gRPC server
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	// RegisterAuthUrlServiceServer
	pb.RegisterHealthServiceServer(s, &Server{}) //
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("Server exited with error")
		}
	}()
	// HTTP server
	mux := http.NewServeMux()

	httpServer = httptest.NewUnstartedServer(mux)
	httpServer.Config.ReadHeaderTimeout = 10 * time.Second
	httpServer.Start()
	// It would be nice if we could Close() the httpServer, but we leak it in the test instead
}

func newDefaultServer(t *testing.T, mockStore *mockdb.MockStore, opts ...ServerOption) (*Server, events.Interface) {
	t.Helper()

	evt, err := events.Setup(context.Background(), &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
	})
	require.NoError(t, err, "failed to setup eventer")

	var c *serverconfig.Config
	tokenKeyPath := generateTokenKey(t)
	c = &serverconfig.Config{
		Auth: serverconfig.AuthConfig{
			TokenKey: tokenKeyPath,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockJwt := mockjwt.NewMockJwtValidator(ctrl)

	server, err := NewServer(
		mockStore,
		evt,
		c,
		mockJwt,
		&ratecache.NoopRestClientCache{},
		providers.NewProviderStore(mockStore),
		opts...,
	)
	require.NoError(t, err, "failed to create server")
	return server, evt
}

func generateTokenKey(t *testing.T) string {
	t.Helper()

	tmpdir := t.TempDir()

	tokenKeyPath := filepath.Join(tmpdir, "/token_key")

	// Write token key to file
	err := os.WriteFile(tokenKeyPath, []byte("test"), 0600)
	require.NoError(t, err, "failed to write token key to file")

	return tokenKeyPath
}

func TestWebhook(t *testing.T) {
	t.Parallel()

	resp, err := http.Get(httpServer.URL + "/api/v1/github/hook")
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}
	defer resp.Body.Close()
}
