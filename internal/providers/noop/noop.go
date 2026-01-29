// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package noop provides a no-op provider implementation.
package noop

import (
	"context"

	"google.golang.org/protobuf/reflect/protoreflect"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// Provider is a no-op provider implementation
// This is useful for testing.
type Provider struct{}

// CanImplement implements the Provider interface
func (*Provider) CanImplement(_ minderv1.ProviderType) bool {
	return false
}

// FetchAllProperties implements the Provider interface
func (*Provider) FetchAllProperties(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	return nil, nil
}

// FetchProperty Implements the Provider interface
func (*Provider) FetchProperty(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}

// GetEntityName implements the Provider interface
func (*Provider) GetEntityName(_ minderv1.Entity, _ *properties.Properties) (string, error) {
	return "", nil
}

// SupportsEntity implements the Provider interface
func (*Provider) SupportsEntity(_ minderv1.Entity) bool {
	return false
}

// CreationOptions implements the Provider interface
func (*Provider) CreationOptions(_ minderv1.Entity) *provifv1.EntityCreationOptions {
	// No-op provider doesn't support any entities
	return nil
}

// RegisterEntity implements the Provider interface
func (*Provider) RegisterEntity(
	_ context.Context, _ minderv1.Entity, props *properties.Properties) (*properties.Properties, error) {
	return props, nil
}

// DeregisterEntity implements the Provider interface
func (*Provider) DeregisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) error {
	return nil
}

// PropertiesToProtoMessage implements the Provider interface
func (*Provider) PropertiesToProtoMessage(_ minderv1.Entity, _ *properties.Properties) (protoreflect.ProtoMessage, error) {
	return nil, nil
}
