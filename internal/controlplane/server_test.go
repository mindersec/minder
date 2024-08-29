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
	"crypto/rand"
	"encoding/base64"
	"io"
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
	"github.com/stacklok/minder/internal/auth"
	mockjwt "github.com/stacklok/minder/internal/auth/jwt/mock"
	mockauthz "github.com/stacklok/minder/internal/authz/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers"
	ghclient "github.com/stacklok/minder/internal/providers/github/clients"
	ghService "github.com/stacklok/minder/internal/providers/github/service"
	mock_github "github.com/stacklok/minder/internal/repositories/github/mock"
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

func newDefaultServer(
	t *testing.T,
	mockStore *mockdb.MockStore,
	mockRepoSvc *mock_github.MockRepositoryService,
	ghClientFactory ghclient.GitHubClientFactory,
) (*Server, events.Interface) {
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
	mockJwt := mockjwt.NewMockValidator(ctrl)

	// Needed to keep these tests working as-is.
	// In future, beef up unit test coverage in the dependencies
	// of this code, and refactor these tests to use stubs.
	eng, err := crypto.NewEngineFromConfig(c)
	require.NoError(t, err)
	ghClientService := ghService.NewGithubProviderService(
		mockStore,
		eng,
		metrics.NewNoopMetrics(),
		// These nil dependencies do not matter for the current tests
		&serverconfig.ProviderConfig{
			GitHubApp: &serverconfig.GitHubAppConfig{
				WebhookSecret: "test",
			},
		},
		nil,
		ghClientFactory,
	)

	server := &Server{
		store:         mockStore,
		evt:           evt,
		cfg:           c,
		mt:            metrics.NewNoopMetrics(),
		jwt:           mockJwt,
		authzClient:   &mockauthz.SimpleClient{},
		idClient:      &auth.IdentityClient{},
		ghProviders:   ghClientService,
		providerStore: providers.NewProviderStore(mockStore),
		repos:         mockRepoSvc,
		cryptoEngine:  eng,
	}

	return server, evt
}

func generateTokenKey(t *testing.T) string {
	t.Helper()

	tmpdir := t.TempDir()

	tokenKeyPath := filepath.Join(tmpdir, "/token_key")

	// generate 256-bit key
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	require.NoError(t, err)
	encodedKey := base64.StdEncoding.EncodeToString(key)

	// Write token key to file
	err = os.WriteFile(tokenKeyPath, []byte(encodedKey), 0600)
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
