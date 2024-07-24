//
// Copyright 2024 Stacklok, Inc.
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

package authz_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	fgasdk "github.com/openfga/go-sdk"
	"github.com/openfga/openfga/cmd/run"
	"github.com/openfga/openfga/pkg/logger"
	"github.com/openfga/openfga/pkg/testutils"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/auth/jwt"
	"github.com/stacklok/minder/internal/authz"
	srvconfig "github.com/stacklok/minder/internal/config/server"
)

var (
	// Re-define authzModel variable since we don't want to export it
	//
	//go:embed model/minder.generated.json
	authzModel string

	// fgaServerRunMux is a mutex to ensure that we start an OpenFGA server
	// at a time. The problem is that OpenFGA server uses global state and
	// if we start multiple servers at the same time, they will conflict
	// with each other.
	fgaServerRunMux = &sync.Mutex{}
)

func TestAllRolesExistInFGAModel(t *testing.T) {
	t.Parallel()

	var m fgasdk.WriteAuthorizationModelRequest
	require.NoError(t, json.Unmarshal([]byte(authzModel), &m), "failed to unmarshal authz model")

	var projectTypeDef fgasdk.TypeDefinition
	var typedeffound bool
	for _, td := range m.TypeDefinitions {
		if td.Type == "project" {
			projectTypeDef = td
			typedeffound = true
			break
		}
	}

	require.True(t, typedeffound, "project type definition not found in authz model")

	t.Logf("relations: %v", projectTypeDef.Relations)

	for r := range authz.AllRolesDescriptions {
		assert.Contains(t, *projectTypeDef.Relations, r.String(), "role %s not found in authz model", r)
	}
}

func TestMigration(t *testing.T) {
	t.Parallel()

	c, stopFunc := newOpenFGAServerAndClient(t)
	defer stopFunc()
	assert.NotNil(t, c)

	assert.NoError(t, c.MigrateUp(context.Background()), "failed to migrate up")
}

func TestVerifyOneProject(t *testing.T) {
	t.Parallel()

	c, stopFunc := newOpenFGAServerAndClient(t)
	defer stopFunc()
	assert.NotNil(t, c)

	ctx := context.Background()

	assert.NoError(t, c.MigrateUp(ctx), "failed to migrate up")

	// this is required to auto-detect the generated model and store
	assert.NoError(t, c.PrepareForRun(ctx), "failed to prepare for run")

	// create a project
	prj := uuid.New()
	assert.NoError(t, c.Write(ctx, "user-1", authz.RoleAdmin, prj), "failed to write project")

	userJWT := openid.New()
	assert.NoError(t, userJWT.Set("sub", "user-1"))
	userctx := jwt.WithAuthTokenContext(ctx, userJWT)

	// verify the project
	assert.NoError(t, c.Check(userctx, "get", prj), "failed to check project")

	// ensure projects for user returns the project
	projects, err := c.ProjectsForUser(userctx, "user-1")
	assert.NoError(t, err, "failed to get projects for user")
	assert.Len(t, projects, 1, "expected 1 project for user")
	assert.Equal(t, prj, projects[0], "expected project to be returned")

	// ensure assignments to project returns the user
	assignments, err := c.AssignmentsToProject(userctx, prj)
	assert.NoError(t, err, "failed to get assignments to project")
	assert.Len(t, assignments, 1, "expected 1 assignment to project")
	assert.Equal(t, "user-1", assignments[0].Subject, "expected user to be assigned to project")

	// delete the project
	assert.NoError(t, c.Delete(ctx, "user-1", authz.RoleAdmin, prj), "failed to delete project")

	// verify the project is gone
	assert.Error(t, c.Check(userctx, "get", prj), "expected project to be gone")

	// ensure projects for user returns no projects
	projects, err = c.ProjectsForUser(userctx, "user-1")
	assert.NoError(t, err, "failed to get projects for user")
	assert.Len(t, projects, 0, "expected 0 projects for user")

	// ensure assignments to project returns no assignments
	assignments, err = c.AssignmentsToProject(userctx, prj)
	assert.NoError(t, err, "failed to get assignments to project")
	assert.Len(t, assignments, 0, "expected 0 assignments to project")
}

func TestVerifyMultipleProjects(t *testing.T) {
	t.Parallel()

	c, stopFunc := newOpenFGAServerAndClient(t)
	defer stopFunc()
	assert.NotNil(t, c)

	ctx := context.Background()

	assert.NoError(t, c.MigrateUp(ctx), "failed to migrate up")

	// this is required to auto-detect the generated model and store
	assert.NoError(t, c.PrepareForRun(ctx), "failed to prepare for run")

	// create a project
	prj1 := uuid.New()
	assert.NoError(t, c.Write(ctx, "user-1", authz.RoleAdmin, prj1), "failed to write project")

	user1JWT := openid.New()
	assert.NoError(t, user1JWT.Set("sub", "user-1"))
	userctx := jwt.WithAuthTokenContext(ctx, user1JWT)

	// verify the project
	assert.NoError(t, c.Check(userctx, "get", prj1), "failed to check project")

	// create another project
	prj2 := uuid.New()
	assert.NoError(t, c.Write(ctx, "user-1", authz.RoleViewer, prj2), "failed to write project")

	// verify the project
	assert.NoError(t, c.Check(userctx, "get", prj2), "failed to check project")

	// create an unrelated project
	prj3 := uuid.New()
	assert.NoError(t, c.Write(ctx, "user-2", authz.RoleAdmin, prj3), "failed to write project")

	// verify the project
	user2JWT := openid.New()
	assert.NoError(t, user2JWT.Set("sub", "user-2"))
	assert.NoError(t, c.Check(jwt.WithAuthTokenContext(ctx, user2JWT), "get", prj3), "failed to check project")

	// verify user-1 cannot operate on project 3
	assert.Error(t, c.Check(userctx, "get", prj3), "expected user-1 to not be able to operate on project 3")

	// ensure projects for user returns the projects
	projects, err := c.ProjectsForUser(userctx, "user-1")
	assert.NoError(t, err, "failed to get projects for user")
	assert.Len(t, projects, 2, "expected 2 projects for user")
	assert.Contains(t, projects, prj1, "expected project to be returned")
	assert.Contains(t, projects, prj2, "expected project to be returned")

	// ensure assignments to project returns the user
	assignments, err := c.AssignmentsToProject(userctx, prj1)
	assert.NoError(t, err, "failed to get assignments to project")
	assert.Len(t, assignments, 1, "expected 1 assignment to project")
	assert.Equal(t, "user-1", assignments[0].Subject, "expected user to be assigned to project")

	// ensure assignments to project returns the user
	assignments, err = c.AssignmentsToProject(userctx, prj2)
	assert.NoError(t, err, "failed to get assignments to project")
	assert.Len(t, assignments, 1, "expected 1 assignment to project")
	assert.Equal(t, "user-1", assignments[0].Subject, "expected user to be assigned to project")

	// Delete the user (which also deletes the projects)
	assert.NoError(t, c.DeleteUser(ctx, "user-1"), "failed to delete user")

	// verify the projects are gone
	assert.Error(t, c.Check(userctx, "get", prj1), "expected project to be gone")
	assert.Error(t, c.Check(userctx, "get", prj2), "expected project to be gone")

	// ensure projects for user returns no projects
	projects, err = c.ProjectsForUser(userctx, "user-1")
	assert.NoError(t, err, "failed to get projects for user")
	assert.Len(t, projects, 0, "expected 0 projects for user")

	// ensure assignments to project returns no assignments
	assignments, err = c.AssignmentsToProject(userctx, prj1)
	assert.NoError(t, err, "failed to get assignments to project")
	assert.Len(t, assignments, 0, "expected 0 assignments to project")
}

func newOpenFGAServerAndClient(t *testing.T) (authz.Client, func()) {
	t.Helper()

	fgaServerRunMux.Lock()

	cfg := testutils.MustDefaultConfigWithRandomPorts()
	cfg.Log.Level = "error"
	cfg.Datastore.Engine = "memory"

	loggr := logger.MustNewLogger("text", "error", "ISO8601")
	serverCtx := &run.ServerContext{Logger: loggr}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		err := serverCtx.Run(ctx, cfg)
		require.NoError(t, err)
	}()

	testutils.EnsureServiceHealthy(t, cfg.GRPC.Addr, cfg.HTTP.Addr, nil)

	testw := zerolog.NewTestWriter(t)
	l := zerolog.New(testw)

	c, err := authz.NewAuthzClient(&srvconfig.AuthzConfig{
		ApiUrl:    "http://" + cfg.HTTP.Addr,
		StoreName: "minder",
		Auth: srvconfig.OpenFGAAuth{
			Method: "none",
		},
	}, &l)
	require.NoError(t, err, "failed to create authz client")

	return c, func() {
		cancel()
		fgaServerRunMux.Unlock()
	}
}
