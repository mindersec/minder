// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package cursor provides a way to encode and decode cursors for paginated queries
package cursor

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// cursorDelimiter is the delimiter used to encode/decode cursors
const cursorDelimiter = ","

// RepoCursor is a cursor for listing repositories
type RepoCursor struct {
	ProjectId string
	Provider  string
	RepoId    int64
}

func (c *RepoCursor) String() string {
	if c == nil || c.ProjectId == "" || c.Provider == "" || c.RepoId <= 0 {
		return ""
	}
	key := strings.Join([]string{c.ProjectId, c.Provider, strconv.Itoa(int(c.RepoId))}, cursorDelimiter)
	return EncodeValue(key)
}

// NewRepoCursor creates a new RepoCursor from an encoded cursor
func NewRepoCursor(encodedCursor string) (*RepoCursor, error) {
	if encodedCursor == "" {
		return &RepoCursor{}, nil
	}

	cursor, err := DecodeValue(encodedCursor)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(cursor, cursorDelimiter)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid cursor: %s", encodedCursor)
	}
	parsedRepoId, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, err
	}

	return &RepoCursor{
		ProjectId: parts[0],
		Provider:  parts[1],
		RepoId:    parsedRepoId,
	}, nil
}

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
