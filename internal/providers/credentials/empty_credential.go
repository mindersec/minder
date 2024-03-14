// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package credentials provides the implementations for the credentials
package credentials

import (
	"net/http"

	"github.com/go-git/go-git/v5"

	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
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
