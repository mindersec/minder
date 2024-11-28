// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/auth"
	mockjwt "github.com/mindersec/minder/internal/auth/jwt/mock"
	mockauthz "github.com/mindersec/minder/internal/authz/mock"
	"github.com/mindersec/minder/internal/controlplane/metrics"
	"github.com/mindersec/minder/internal/crypto"
	mock_service "github.com/mindersec/minder/internal/entities/properties/service/mock"
	"github.com/mindersec/minder/internal/providers"
	ghclient "github.com/mindersec/minder/internal/providers/github/clients"
	ghService "github.com/mindersec/minder/internal/providers/github/service"
	mock_reposvc "github.com/mindersec/minder/internal/repositories/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer"
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

// nolint: unparam
func newDefaultServer(
	t *testing.T,
	mockStore *mockdb.MockStore,
	// TODO: can be removed?
	mockRepoSvc *mock_reposvc.MockRepositoryService,
	mockPropSvc *mock_service.MockPropertiesService,
	ghClientFactory ghclient.GitHubClientFactory,
) *Server {
	t.Helper()

	evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
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
		props:         mockPropSvc,
		cryptoEngine:  eng,
	}

	return server
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

func TestHandleRequestTooLarge(t *testing.T) {
	t.Parallel()

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Reading the body to trigger the limit check
		_, err := io.ReadAll(r.Body)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				// this is the expected error
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			// this is an unexpected error
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// a 200 isn't necessary, but it would fail the test if the handler didn't return 413
		w.WriteHeader(http.StatusOK)
	}

	ts := httptest.NewServer(withMaxSizeMiddleware(http.HandlerFunc(handler)))

	event := github.PackageEvent{
		Action: github.String("published"),
		Repo: &github.Repository{
			ID:   github.Int64(12345),
			Name: github.String("stacklok/minder"),
		},
		Org: &github.Organization{
			Login: github.String("stacklok"),
		},
	}
	packageJson, err := json.Marshal(event)
	require.NoError(t, err, "failed to marshal package event")

	maliciousBody := strings.NewReader(strings.Repeat("1337", 100000000))
	maliciousBodyReader := io.MultiReader(maliciousBody, maliciousBody, maliciousBody, maliciousBody, maliciousBody)
	_ = packageJson

	req, err := http.NewRequest("POST", ts.URL, maliciousBodyReader)
	require.NoError(t, err, "failed to create request")

	req.Header.Add("X-GitHub-Event", "meta")
	req.Header.Add("X-GitHub-Delivery", "12345")
	req.Header.Add("Content-Type", "application/json")
	resp, err := httpDoWithRetry(ts.Client(), req)
	require.NoError(t, err, "failed to make request")
	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode, "unexpected status code")
}

func httpDoWithRetry(client *http.Client, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	err := backoff.Retry(func() error {
		var err error
		resp, err = client.Do(req)
		return err
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Second), 3))

	return resp, err
}
