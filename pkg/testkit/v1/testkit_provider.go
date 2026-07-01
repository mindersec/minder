// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"google.golang.org/protobuf/reflect/protoreflect"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provv1 "github.com/mindersec/minder/pkg/providers/v1"
)

var (
	// ErrNotIngesterOverridden is returned when a provider trait is not overridden.
	ErrNotIngesterOverridden = errors.New("ingester not overridden")
)

// Ensure that TestKit implements the Provider interface
var _ provv1.Provider = &TestKit{}

// CanImplement implements the Provider interface.
// It returns true since we don't have any restrictions on the provider.
func (*TestKit) CanImplement(_ minderv1.ProviderType) bool {
	return true
}

// FetchAllProperties implements the Provider interface.
func (*TestKit) FetchAllProperties(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	return nil, nil
}

// FetchProperty implements the Provider interface.
func (*TestKit) FetchProperty(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}

// GetEntityName implements the Provider interface.
func (*TestKit) GetEntityName(_ minderv1.Entity, _ *properties.Properties) (string, error) {
	return "", nil
}

// SupportsEntity implements the Provider interface.
func (*TestKit) SupportsEntity(_ minderv1.Entity) bool {
	return true
}

// ProviderClassInfo implements the Provider interface.
func (*TestKit) ProviderClassInfo() *minderv1.ProviderClassInfo {
	return nil
}

// CreationOptions implements the Provider interface.
func (*TestKit) CreationOptions(_ minderv1.Entity) *provv1.EntityCreationOptions {
	// Test scaffold returns no-op options
	return &provv1.EntityCreationOptions{
		RegisterWithProvider:       false,
		PublishReconciliationEvent: false,
	}
}

// RegisterEntity implements the Provider interface.
func (*TestKit) RegisterEntity(_ context.Context, _ minderv1.Entity, props *properties.Properties,
) (*properties.Properties, error) {
	// Since this is a test scaffold, we accept all entity types
	return props, nil
}

// DeregisterEntity implements the Provider interface.
func (*TestKit) DeregisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) error {
	return nil
}

// PropertiesToProtoMessage implements the Provider interface.
func (*TestKit) PropertiesToProtoMessage(_ minderv1.Entity, _ *properties.Properties) (protoreflect.ProtoMessage, error) {
	return nil, nil
}

// Clone Implements the Git trait. This initializes an in-memory repository with the mocked filesystem if provided.
func (tk *TestKit) Clone(_ context.Context, _ string, _ string) (*git.Repository, error) {
	if len(tk.mockFS) == 0 {
		return nil, ErrNotIngesterOverridden
	}

	storer := memory.NewStorage()
	fs := memfs.New()

	for path, content := range tk.mockFS {
		if err := fs.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, err
		}
		f, err := fs.Create(path)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write([]byte(content)); err != nil {
			_ = f.Close()
			return nil, err
		}
		_ = f.Close()
	}

	repo, err := git.Init(storer, fs)
	if err != nil {
		return nil, err
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	for path := range tk.mockFS {
		if _, err := w.Add(path); err != nil {
			return nil, err
		}
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "TestKit",
			Email: "testkit@minder.test",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	return repo, nil
}
