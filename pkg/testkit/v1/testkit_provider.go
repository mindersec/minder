// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"errors"

	"github.com/go-git/go-git/v5"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/entities/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provv1 "github.com/mindersec/minder/pkg/providers/v1"
)

var (
	// ErrNotIngeserOverridden is returned when a provider trait is not overridden.
	ErrNotIngeserOverridden = errors.New("ingester not overridden")
)

// Ensure that TestKit implements the Provider interface
var _ provv1.Provider = &TestKit{}

// CanImplement implements the Provider interface.
// It returns true since we don't have any restrictions on the provider.
func (_ *TestKit) CanImplement(_ minderv1.ProviderType) bool {
	return true
}

// FetchAllProperties implements the Provider interface.
func (_ *TestKit) FetchAllProperties(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	return nil, nil
}

// FetchProperty implements the Provider interface.
func (_ *TestKit) FetchProperty(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}

// GetEntityName implements the Provider interface.
func (_ *TestKit) GetEntityName(_ minderv1.Entity, _ *properties.Properties) (string, error) {
	return "", nil
}

// SupportsEntity implements the Provider interface.
func (_ *TestKit) SupportsEntity(_ minderv1.Entity) bool {
	return true
}

// RegisterEntity implements the Provider interface.
func (_ *TestKit) RegisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) (*properties.Properties, error) {
	return nil, nil
}

// DeregisterEntity implements the Provider interface.
func (_ *TestKit) DeregisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) error {
	return nil
}

// ReregisterEntity implements the Provider interface.
func (_ *TestKit) ReregisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) error {
	return nil
}

// PropertiesToProtoMessage implements the Provider interface.
func (_ *TestKit) PropertiesToProtoMessage(_ minderv1.Entity, _ *properties.Properties) (protoreflect.ProtoMessage, error) {
	return nil, nil
}

// Clone Implements the Git trait. This is a stub implementation that allows us to instantiate a Git ingester.
// This will later be overridden by the actual implementation.
func (_ *TestKit) Clone(_ context.Context, _ string, _ string) (*git.Repository, error) {
	// Note that this should not be called. If it is, it means that the ingester has not been overridden.
	return nil, ErrNotIngeserOverridden
}
