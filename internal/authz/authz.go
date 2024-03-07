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
	"strings"

	"github.com/google/uuid"
	fgasdk "github.com/openfga/go-sdk"
	fgaclient "github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
	"github.com/rs/zerolog"
	"k8s.io/client-go/transport"

	"github.com/stacklok/minder/internal/auth"
	srvconfig "github.com/stacklok/minder/internal/config/server"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
	l   *zerolog.Logger
}

var _ Client = &ClientWrapper{}

// NewAuthzClient returns a new AuthzClientWrapper
func NewAuthzClient(cfg *srvconfig.AuthzConfig, l *zerolog.Logger) (Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cliWrap := &ClientWrapper{
		cfg: cfg,
		l:   l,
	}

	if err := cliWrap.initAuthzClient(); err != nil {
		return nil, err
	}

	return cliWrap, nil
}

// initAuthzClient initializes the authz client based on the configuration.
// Note that this assumes the configuration has already been validated.
func (a *ClientWrapper) initAuthzClient() error {
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
	storeID, err := a.findStoreByName(ctx)
	if err != nil {
		return fmt.Errorf("unable to find authz store: %w", err)
	}

	a.cli.SetStoreId(storeID)

	modelID, err := a.findLatestModel(ctx)
	if err != nil {
		return fmt.Errorf("unable to find authz model: %w", err)
	}

	if err := a.cli.SetAuthorizationModelId(modelID); err != nil {
		return fmt.Errorf("unable to set authz model: %w", err)
	}

	return nil
}

// StoreIDProvided returns true if the store ID was provided in the configuration
func (a *ClientWrapper) StoreIDProvided() bool {
	return a.cfg.StoreID != ""
}

// MigrateUp runs the authz migrations. For OpenFGA this means creating the store
// and writing the authz model.
func (a *ClientWrapper) MigrateUp(ctx context.Context) error {
	if !a.StoreIDProvided() {
		if err := a.ensureAuthzStore(ctx); err != nil {
			return err
		}
	}

	m, err := a.writeModel(ctx)
	if err != nil {
		return fmt.Errorf("error while writing authz model: %w", err)
	}

	a.l.Printf("Wrote authz model %s\n", m)

	return nil
}

func (a *ClientWrapper) ensureAuthzStore(ctx context.Context) error {
	storeName := a.cfg.StoreName
	storeID, err := a.findStoreByName(ctx)
	if err != nil && !errors.Is(err, ErrStoreNotFound) {
		return err
	} else if errors.Is(err, ErrStoreNotFound) {
		a.l.Printf("Creating authz store %s\n", storeName)
		id, err := a.createStore(ctx)
		if err != nil {
			return err
		}
		a.l.Printf("Created authz store %s/%s\n", id, storeName)
		a.cli.SetStoreId(id)
		return nil
	}

	a.l.Printf("Not creating store. Found store with name '%s' and ID '%s'.\n",
		storeName, storeID)

	a.cli.SetStoreId(storeID)
	return nil
}

// findStoreByName returns the store ID for the configured store name
func (a *ClientWrapper) findStoreByName(ctx context.Context) (string, error) {
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

// createStore creates a new store with the configured name
func (a *ClientWrapper) createStore(ctx context.Context) (string, error) {
	st, err := a.cli.CreateStore(ctx).Body(fgaclient.ClientCreateStoreRequest{
		Name: a.cfg.StoreName,
	}).Execute()
	if err != nil {
		return "", fmt.Errorf("error while creating authz store: %w", err)
	}

	return st.Id, nil
}

// findLatestModel returns the latest authz model ID
func (a *ClientWrapper) findLatestModel(ctx context.Context) (string, error) {
	resp, err := a.cli.ReadLatestAuthorizationModel(ctx).Execute()
	if err != nil {
		return "", fmt.Errorf("error while reading authz model: %w", err)
	}

	return resp.AuthorizationModel.Id, nil
}

// writeModel writes the authz model to the configured store
func (a *ClientWrapper) writeModel(ctx context.Context) (string, error) {
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
	return a.write(ctx, fgasdk.TupleKey{
		User:     getUserForTuple(user),
		Relation: role.String(),
		Object:   getProjectForTuple(project),
	})
}

// Adopt writes a relationship between the parent and child projects
func (a *ClientWrapper) Adopt(ctx context.Context, parent, child uuid.UUID) error {
	return a.write(ctx, fgasdk.TupleKey{
		User:     getProjectForTuple(parent),
		Relation: "parent",
		Object:   getProjectForTuple(child),
	})
}

func (a *ClientWrapper) write(ctx context.Context, t fgasdk.TupleKey) error {
	resp, err := a.cli.WriteTuples(ctx).Options(fgaclient.ClientWriteOptions{}).
		Body([]fgasdk.TupleKey{t}).Execute()
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return nil
	} else if err != nil {
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
	return a.doDelete(ctx, getUserForTuple(user), role.String(), getProjectForTuple(project))
}

// Orphan removes the relationship between the parent and child projects
func (a *ClientWrapper) Orphan(ctx context.Context, parent, child uuid.UUID) error {
	return a.doDelete(ctx, getProjectForTuple(parent), "parent", getProjectForTuple(child))
}

// doDelete wraps the OpenFGA DeleteTuples call and handles edge cases as needed. It takes
// the user, role, and project as tuple-formatted strings.
func (a *ClientWrapper) doDelete(ctx context.Context, user string, role string, project string) error {
	resp, err := a.cli.DeleteTuples(ctx).Options(fgaclient.ClientWriteOptions{}).Body([]fgasdk.TupleKeyWithoutCondition{
		{
			User:     user,
			Relation: role,
			Object:   project,
		},
	}).Execute()
	if err != nil && strings.Contains(err.Error(), "cannot delete a tuple which does not exist") {
		return nil
	} else if err != nil {
		return fmt.Errorf("unable to remove authorization tuple: %w", err)
	}

	for _, w := range resp.Deletes {
		if w.Error != nil {
			return fmt.Errorf("unable to remove authorization tuple: %w", w.Error)
		}
	}

	return nil
}

// DeleteUser removes all tuples for the given user
func (a *ClientWrapper) DeleteUser(ctx context.Context, user string) error {
	for role := range AllRoles {
		listresp, err := a.cli.ListObjects(ctx).Body(fgaclient.ClientListObjectsRequest{
			Type:     "project",
			Relation: role.String(),
			User:     getUserForTuple(user),
		}).Execute()
		if err != nil {
			return fmt.Errorf("unable to list authorization tuples: %w", err)
		}

		for _, obj := range listresp.GetObjects() {
			if err := a.doDelete(ctx, getUserForTuple(user), role.String(), obj); err != nil {
				return err
			}
		}
	}

	return nil
}

// AssignmentsToProject lists the current role assignments that are scoped to a project
func (a *ClientWrapper) AssignmentsToProject(ctx context.Context, project uuid.UUID) ([]*minderv1.RoleAssignment, error) {
	o := getProjectForTuple(project)
	prjStr := project.String()

	var pagesize int32 = 50
	var contTok *string = nil

	assignments := []*minderv1.RoleAssignment{}

	for {
		resp, err := a.cli.Read(ctx).Options(fgaclient.ClientReadOptions{
			PageSize:          &pagesize,
			ContinuationToken: contTok,
		}).Body(fgaclient.ClientReadRequest{
			Object: &o,
		}).Execute()
		if err != nil {
			return nil, fmt.Errorf("unable to read authorization tuples: %w", err)
		}

		for _, t := range resp.GetTuples() {
			k := t.GetKey()
			r, err := ParseRole(k.GetRelation())
			if err != nil {
				a.l.Err(err).Msg("Found invalid role in authz store")
				continue
			}
			assignments = append(assignments, &minderv1.RoleAssignment{
				Subject: getUserFromTuple(k.GetUser()),
				Role:    r.String(),
				Project: &prjStr,
			})
		}

		if resp.GetContinuationToken() == "" {
			break
		}

		contTok = &resp.ContinuationToken
	}

	return assignments, nil
}

// ProjectsForUser lists the projects that the given user has access to
func (a *ClientWrapper) ProjectsForUser(ctx context.Context, sub string) ([]uuid.UUID, error) {
	u := getUserForTuple(sub)

	var pagesize int32 = 50
	var contTok *string = nil

	projs := map[string]any{}
	projectObj := "project:"

	for {
		resp, err := a.cli.Read(ctx).Options(fgaclient.ClientReadOptions{
			PageSize:          &pagesize,
			ContinuationToken: contTok,
		}).Body(fgaclient.ClientReadRequest{
			User:   &u,
			Object: &projectObj,
		}).Execute()
		if err != nil {
			return nil, fmt.Errorf("unable to read authorization tuples: %w", err)
		}

		for _, t := range resp.GetTuples() {
			k := t.GetKey()

			projs[k.GetObject()] = struct{}{}
		}

		if resp.GetContinuationToken() == "" {
			break
		}

		contTok = &resp.ContinuationToken
	}

	out := []uuid.UUID{}
	for proj := range projs {
		u, err := uuid.Parse(getProjectFromTuple(proj))
		if err != nil {
			continue
		}

		out = append(out, u)

		children, err := a.traverseProjectsForParent(ctx, u)
		if err != nil {
			return nil, err
		}

		out = append(out, children...)
	}

	return out, nil
}

// traverseProjectsForParent is a recursive function that traverses the project
// hierarchy to find all projects that the parent project has access to.
func (a *ClientWrapper) traverseProjectsForParent(ctx context.Context, parent uuid.UUID) ([]uuid.UUID, error) {
	projects := []uuid.UUID{}

	resp, err := a.cli.ListObjects(ctx).Body(fgaclient.ClientListObjectsRequest{
		User:     getProjectForTuple(parent),
		Relation: "parent",
		Type:     "project",
	}).Execute()

	if err != nil {
		return nil, fmt.Errorf("unable to read authorization tuples: %w", err)
	}

	for _, obj := range resp.GetObjects() {
		u, err := uuid.Parse(getProjectFromTuple(obj))
		if err != nil {
			continue
		}
		projects = append(projects, u)
	}

	for _, proj := range projects {
		children, err := a.traverseProjectsForParent(ctx, proj)
		if err != nil {
			return nil, err
		}
		projects = append(projects, children...)
	}

	return projects, nil
}

func getUserForTuple(user string) string {
	return "user:" + user
}

func getProjectForTuple(project uuid.UUID) string {
	return "project:" + project.String()
}

func getUserFromTuple(user string) string {
	return strings.TrimPrefix(user, "user:")
}

func getProjectFromTuple(project string) string {
	return strings.TrimPrefix(project, "project:")
}
