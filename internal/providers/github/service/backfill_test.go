// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/db/embedded"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestBackfillOrganizations_NoProviders(t *testing.T) {
	t.Parallel()

	store, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	require.NoError(t, err)

	// No providers in the DB, so backfill should succeed and do nothing
	err = BackfillOrganizations(context.Background(), store)
	require.NoError(t, err)
}

func TestBackfillOrganizations_CreatesEntity(t *testing.T) {
	t.Parallel()

	store, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	require.NoError(t, err)

	// Create a project first
	proj, err := store.CreateProject(context.Background(), db.CreateProjectParams{
		Name:     "test-backfill",
		Metadata: []byte(`{}`),
	})
	require.NoError(t, err)

	// Create a GitHub App provider
	prov, err := store.CreateProvider(context.Background(), db.CreateProviderParams{
		Name:       "github-app-test-org",
		ProjectID:  proj.ID,
		Class:      db.ProviderClassGithubApp,
		Implements: []db.ProviderType{db.ProviderTypeGithub, db.ProviderTypeGit},
		AuthFlows:  []db.AuthorizationFlow{db.AuthorizationFlowUserInput},
		Definition: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	// Run the backfill
	err = BackfillOrganizations(context.Background(), store)
	require.NoError(t, err)

	// Verify organization entity was created
	ent, err := store.GetEntityByName(context.Background(), db.GetEntityByNameParams{
		EntityType: db.EntitiesOrganization,
		Name:       "test-org",
		ProviderID: prov.ID,
		ProjectID:  proj.ID,
	})
	require.NoError(t, err)
	require.Equal(t, "test-org", ent.Name)
	require.Equal(t, db.EntitiesOrganization, ent.EntityType)

	// Verify property was set
	prop, err := store.GetProperty(context.Background(), db.GetPropertyParams{
		EntityID: ent.ID,
		Key:      properties.PropertyName,
	})
	require.NoError(t, err)

	val, err := db.PropValueFromDbV1(prop.Value)
	require.NoError(t, err)
	require.Equal(t, "test-org", val)
}

func TestBackfillOrganizations_Idempotent(t *testing.T) {
	t.Parallel()

	store, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	require.NoError(t, err)

	// Create a project
	proj, err := store.CreateProject(context.Background(), db.CreateProjectParams{
		Name:     "test-backfill-idempotent",
		Metadata: []byte(`{}`),
	})
	require.NoError(t, err)

	// Create a GitHub App provider
	_, err = store.CreateProvider(context.Background(), db.CreateProviderParams{
		Name:       "github-app-my-idempotent-org",
		ProjectID:  proj.ID,
		Class:      db.ProviderClassGithubApp,
		Implements: []db.ProviderType{db.ProviderTypeGithub, db.ProviderTypeGit},
		AuthFlows:  []db.AuthorizationFlow{db.AuthorizationFlowUserInput},
		Definition: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	// Run backfill twice - second time should not fail
	err = BackfillOrganizations(context.Background(), store)
	require.NoError(t, err)

	err = BackfillOrganizations(context.Background(), store)
	require.NoError(t, err)
}

func TestGitHubProviderFacet(t *testing.T) {
	t.Parallel()

	facet := &GitHubProviderFacet{
		Provider: &db.Provider{
			Name:  "github-app-my-org",
			Class: db.ProviderClassGithubApp,
		},
		InstallationOwner: "my-org",
	}

	require.NotNil(t, facet.Provider)
	require.Equal(t, "my-org", facet.InstallationOwner)
	require.Equal(t, "github-app-my-org", facet.Provider.Name)
	require.Equal(t, db.ProviderClassGithubApp, facet.Provider.Class)
}
