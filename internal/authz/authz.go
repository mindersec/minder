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

// Package authz provides the authorization utilities for minder
package authz

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	fgasdk "github.com/openfga/go-sdk"
	fgaclient "github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
	"k8s.io/client-go/transport"

	"github.com/stacklok/minder/internal/auth"
	srvconfig "github.com/stacklok/minder/internal/config/server"
)

var (
	// ErrStoreNotFound denotes the error where the store wasn't found via the
	// given configuration.
	ErrStoreNotFound = errors.New("Store not found")

	//go:embed model/minder.generated.json
	authzModel string
)

// ClientWrapper is a wrapper for the OpenFgaClient.
// It is used to provide a common interface for the client and a way to
// refresh authentication to the authz provider when needed.
type ClientWrapper struct {
	cfg *srvconfig.AuthzConfig
	cli *fgaclient.OpenFgaClient
}

var _ Client = &ClientWrapper{}

// NewAuthzClient returns a new AuthzClientWrapper
func NewAuthzClient(cfg *srvconfig.AuthzConfig) (*ClientWrapper, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cliWrap := &ClientWrapper{
		cfg: cfg,
	}

	if err := cliWrap.ResetAuthzClient(); err != nil {
		return nil, err
	}

	return cliWrap, nil
}

// ResetAuthzClient initializes the authz client based on the configuration.
// Note that this assumes the configuration has already been validated.
func (a *ClientWrapper) ResetAuthzClient() error {
	clicfg := &fgaclient.ClientConfiguration{
		ApiUrl: a.cfg.ApiUrl,
		Credentials: &credentials.Credentials{
			// We use our own bearer auth round tripper so we can refresh the token
			Method: credentials.CredentialsMethodNone,
		},
	}

	if a.cfg.StoreID != "" {
		clicfg.StoreId = a.cfg.StoreID
	}

	if a.cfg.Auth.Method == "token" {
		rt, err := transport.NewBearerAuthWithRefreshRoundTripper("", a.cfg.Auth.Token.TokenPath, http.DefaultTransport)
		if err != nil {
			return fmt.Errorf("failed to create bearer auth round tripper: %w", err)
		}

		clicfg.HTTPClient = &http.Client{
			Transport: rt,
		}
	}

	cli, err := fgaclient.NewSdkClient(clicfg)
	if err != nil {
		return fmt.Errorf("failed to create SDK client: %w", err)
	}

	a.cli = cli
	return nil
}

// PrepareForRun initializes the authz client based on the configuration.
// This is handy when migrations have already been done and helps us auto-discover
// the store ID and model.
func (a *ClientWrapper) PrepareForRun(ctx context.Context) error {
	storeID, err := a.FindStoreByName(ctx)
	if err != nil {
		return fmt.Errorf("unable to find authz store: %w", err)
	}

	a.cli.SetStoreId(storeID)

	modelID, err := a.FindLatestModel(ctx)
	if err != nil {
		return fmt.Errorf("unable to find authz model: %w", err)
	}

	if err := a.cli.SetAuthorizationModelId(modelID); err != nil {
		return fmt.Errorf("unable to set authz model: %w", err)
	}

	return nil
}

// GetClient returns the OpenFgaClient
func (a *ClientWrapper) GetClient() *fgaclient.OpenFgaClient {
	// TODO: check if token is expired and refresh it
	// Note that this will probably need a mutex
	return a.cli
}

// GetConfig returns the authz configuration used to build the client
func (a *ClientWrapper) GetConfig() *srvconfig.AuthzConfig {
	return a.cfg
}

// StoreIDProvided returns true if the store ID was provided in the configuration
func (a *ClientWrapper) StoreIDProvided() bool {
	return a.cfg.StoreID != ""
}

// FindStoreByName returns the store ID for the configured store name
func (a *ClientWrapper) FindStoreByName(ctx context.Context) (string, error) {
	stores, err := a.cli.ListStores(ctx).Execute()
	if err != nil {
		return "", fmt.Errorf("error while listing authz stores: %w", err)
	}

	// TODO: We might want to handle pagination here.
	for _, store := range stores.Stores {
		if store.Name == a.cfg.StoreName {
			return store.Id, nil
		}
	}

	return "", ErrStoreNotFound
}

// CreateStore creates a new store with the configured name
func (a *ClientWrapper) CreateStore(ctx context.Context) (string, error) {
	st, err := a.cli.CreateStore(ctx).Body(fgaclient.ClientCreateStoreRequest{
		Name: a.cfg.StoreName,
	}).Execute()
	if err != nil {
		return "", fmt.Errorf("error while creating authz store: %w", err)
	}

	return st.Id, nil
}

// FindLatestModel returns the latest authz model ID
func (a *ClientWrapper) FindLatestModel(ctx context.Context) (string, error) {
	resp, err := a.cli.ReadLatestAuthorizationModel(ctx).Execute()
	if err != nil {
		return "", fmt.Errorf("error while reading authz model: %w", err)
	}

	return resp.AuthorizationModel.Id, nil
}

// WriteModel writes the authz model to the configured store
func (a *ClientWrapper) WriteModel(ctx context.Context) (string, error) {
	var body fgasdk.WriteAuthorizationModelRequest
	if err := json.Unmarshal([]byte(authzModel), &body); err != nil {
		return "", fmt.Errorf("failed to unmarshal authz model: %w", err)
	}

	data, err := a.cli.WriteAuthorizationModel(ctx).Body(body).Execute()
	if err != nil {
		return "", fmt.Errorf("error while writing authz model: %w", err)
	}

	return data.GetAuthorizationModelId(), nil
}

// Check checks if the user is authorized to perform the given action on the
// given project.
func (a *ClientWrapper) Check(ctx context.Context, action string, project uuid.UUID) error {
	// TODO: set ClientCheckOptions like in
	// https://openfga.dev/docs/getting-started/perform-check#02-calling-check-api
	options := fgaclient.ClientCheckOptions{}
	userString := getUserForTuple(auth.GetUserSubjectFromContext(ctx))
	body := fgaclient.ClientCheckRequest{
		User:     userString,
		Relation: action,
		Object:   getProjectForTuple(project),
	}
	result, err := a.cli.Check(ctx).Options(options).Body(body).Execute()
	if err != nil {
		return fmt.Errorf("OpenFGA error: %w", err)
	}
	if result.Allowed != nil && *result.Allowed {
		return nil
	}
	return ErrNotAuthorized
}

// Write persists the given role for the given user and project
func (a *ClientWrapper) Write(ctx context.Context, user string, role Role, project uuid.UUID) error {
	resp, err := a.cli.WriteTuples(ctx).Options(fgaclient.ClientWriteOptions{}).Body([]fgasdk.TupleKey{
		{
			User:     getUserForTuple(user),
			Relation: role.String(),
			Object:   getProjectForTuple(project),
		},
	}).Execute()
	if err != nil {
		return fmt.Errorf("unable to persist authorization tuple: %w", err)
	}

	for _, w := range resp.Writes {
		if w.Error != nil {
			return fmt.Errorf("unable to persist authorization tuple: %w", w.Error)
		}
	}

	return nil
}

// Delete removes the given role for the given user and project
func (a *ClientWrapper) Delete(ctx context.Context, user string, role Role, project uuid.UUID) error {
	resp, err := a.cli.DeleteTuples(ctx).Options(fgaclient.ClientWriteOptions{}).Body([]fgasdk.TupleKeyWithoutCondition{
		{
			User:     getUserForTuple(user),
			Relation: role.String(),
			Object:   getProjectForTuple(project),
		},
	}).Execute()
	if err != nil {
		return fmt.Errorf("unable to remove authorization tuple: %w", err)
	}

	for _, w := range resp.Deletes {
		if w.Error != nil {
			return fmt.Errorf("unable to remove authorization tuple: %w", w.Error)
		}
	}

	return nil
}

func getUserForTuple(user string) string {
	return "user:" + user
}

func getProjectForTuple(project uuid.UUID) string {
	return "project:" + project.String()
}
