// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestEntityInstanceString_ContainsName(t *testing.T) {
	t.Parallel()
	ei := EntityInstance{
		ID:         uuid.New(),
		Type:       minderv1.Entity_ENTITY_REPOSITORIES,
		Name:       "myorg/myrepo",
		ProviderID: uuid.New(),
		ProjectID:  uuid.New(),
	}
	s := ei.String()
	if !strings.Contains(s, "myorg/myrepo") {
		t.Errorf("EntityInstance.String() = %q, want to contain name", s)
	}
}

func TestEntityInstanceString_ContainsProviderID(t *testing.T) {
	t.Parallel()
	providerID := uuid.New()
	ei := EntityInstance{
		ID:         uuid.New(),
		ProviderID: providerID,
		ProjectID:  uuid.New(),
		Name:       "repo",
	}
	s := ei.String()
	if !strings.Contains(s, providerID.String()) {
		t.Errorf("EntityInstance.String() = %q, want to contain provider id", s)
	}
}

func TestNewEntityWithPropertiesFromInstance_RoundTrip(t *testing.T) {
	t.Parallel()
	ei := EntityInstance{
		ID:        uuid.New(),
		Type:      minderv1.Entity_ENTITY_REPOSITORIES,
		Name:      "test/repo",
		ProjectID: uuid.New(),
	}
	props := properties.NewProperties(map[string]any{"key": "value"})
	ewp := NewEntityWithPropertiesFromInstance(ei, props)
	if ewp.Entity.Name != ei.Name {
		t.Errorf("entity name = %q, want %q", ewp.Entity.Name, ei.Name)
	}
	if ewp.Properties != props {
		t.Error("properties not stored correctly")
	}
}

func TestEntityWithPropertiesUpdateProperties(t *testing.T) {
	t.Parallel()
	ewp := &EntityWithProperties{
		Entity: EntityInstance{Name: "repo"},
	}
	newProps := properties.NewProperties(map[string]any{"updated": true})
	ewp.UpdateProperties(newProps)
	if ewp.Properties != newProps {
		t.Error("UpdateProperties did not update the properties")
	}
}

func TestEntityWithPropertiesNeedsPropertyLoad_FewProps(t *testing.T) {
	t.Parallel()
	ewp := &EntityWithProperties{
		Entity:     EntityInstance{Name: "repo"},
		Properties: properties.NewProperties(map[string]any{"one": 1}),
	}
	if !ewp.NeedsPropertyLoad() {
		t.Error("expected NeedsPropertyLoad() to return true when <= 2 properties")
	}
}

func TestEntityWithPropertiesNeedsPropertyLoad_ManyProps(t *testing.T) {
	t.Parallel()
	ewp := &EntityWithProperties{
		Entity: EntityInstance{Name: "repo"},
		Properties: properties.NewProperties(map[string]any{
			"a": 1, "b": 2, "c": 3,
		}),
	}
	if ewp.NeedsPropertyLoad() {
		t.Error("expected NeedsPropertyLoad() to return false when > 2 properties")
	}
}

func TestEntityWithPropertiesString_ContainsEntityKeyword(t *testing.T) {
	t.Parallel()
	ewp := &EntityWithProperties{
		Entity: EntityInstance{
			ID:   uuid.New(),
			Name: "myrepo",
		},
		Properties: properties.NewProperties(map[string]any{}),
	}
	s := ewp.String()
	if !strings.Contains(s, "ENTITY") {
		t.Errorf("EntityWithProperties.String() = %q, want prefix 'ENTITY'", s)
	}
}
