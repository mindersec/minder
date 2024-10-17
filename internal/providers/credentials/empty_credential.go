// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package credentials provides the implementations for the credentials
package credentials

import (
	"net/http"

	"github.com/go-git/go-git/v5"

	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// EmptyCredential is an empty credential whose operations are no-ops
type EmptyCredential struct {
}

// Ensure that the EmptyCredential implements the GitCredential interface
var _ provifv1.GitCredential = (*EmptyCredential)(nil)

// Ensure that the EmptyCredential implements the RestCredential interface
var _ provifv1.RestCredential = (*EmptyCredential)(nil)

// NewEmptyCredential creates a new EmptyCredential
func NewEmptyCredential() *EmptyCredential {
	return &EmptyCredential{}
}

// SetAuthorizationHeader is a no-op
func (*EmptyCredential) SetAuthorizationHeader(*http.Request) {
}

// AddToPushOptions is a no-op
func (*EmptyCredential) AddToPushOptions(*git.PushOptions, string) {
}

// AddToCloneOptions is a no-op
func (*EmptyCredential) AddToCloneOptions(*git.CloneOptions) {
}
