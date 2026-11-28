// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package cursor provides a way to encode and decode cursors for paginated queries
package cursor

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EncodeValue encodes a string into a base64 encoded string
func EncodeValue(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(value))
}

// DecodeValue decodes a base64 encoded string into a string
func DecodeValue(value string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("error decoding cursor: %w", err)
	}
	return string(decoded), nil
}

// ProviderCursor is the the creation time of the provider
type ProviderCursor struct {
	// CreatedAt is the creation time of the provider
	CreatedAt time.Time
}

// NewProviderCursor creates a new ProviderCursor from an encoded cursor
func NewProviderCursor(encodedCursor string) (*ProviderCursor, error) {
	if encodedCursor == "" {
		return &ProviderCursor{}, nil
	}

	cursor, err := DecodeValue(encodedCursor)
	if err != nil {
		return nil, err
	}

	// parse time with as much precision as possible
	creationTime, err := time.Parse(time.RFC3339Nano, cursor)
	if err != nil {
		return nil, err
	}

	return &ProviderCursor{
		CreatedAt: creationTime,
	}, nil
}

func (c *ProviderCursor) String() string {
	if c == nil {
		return ""
	}
	return EncodeValue(c.CreatedAt.Format(time.RFC3339Nano))
}

// EntityCursor holds the UUID of the last seen entity for keyset pagination.
type EntityCursor struct {
	ID uuid.UUID
}

// NewEntityCursor decodes an encoded cursor string into an EntityCursor.
// An empty string returns a zero cursor (uuid.Nil) meaning start from the beginning.
func NewEntityCursor(encodedCursor string) (*EntityCursor, error) {
	if encodedCursor == "" {
		return &EntityCursor{ID: uuid.Nil}, nil
	}

	raw, err := DecodeValue(encodedCursor)
	if err != nil {
		return nil, fmt.Errorf("error decoding entity cursor: %w", err)
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid entity cursor (not a UUID): %w", err)
	}
	return &EntityCursor{ID: id}, nil
}

// String encodes the EntityCursor to a base64 string. Returns "" for uuid.Nil.
func (c EntityCursor) String() string {
	if c.ID == uuid.Nil {
		return ""
	}
	return EncodeValue(c.ID.String())
}
