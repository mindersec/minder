// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"net/http"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/require"
)

var (
	emptyCredential = NewEmptyCredential()
)

func TestEmptyCredentialSetAuthorizationHeader(t *testing.T) {
	t.Parallel()

	expected := &http.Request{
		Header: http.Header{},
	}
	req := &http.Request{
		Header: http.Header{},
	}
	emptyCredential.SetAuthorizationHeader(req)
	require.Equal(t, expected, req)
}

func TestEmptyCredentialAddToPushOptions(t *testing.T) {
	t.Parallel()

	username := "user"
	expected := &git.PushOptions{}
	pushOptions := &git.PushOptions{}
	emptyCredential.AddToPushOptions(pushOptions, username)
	require.Equal(t, expected, pushOptions)
}

func TestEmptyCredentialAddToCloneOptions(t *testing.T) {
	t.Parallel()

	expected := &git.CloneOptions{}
	cloneOptions := &git.CloneOptions{}
	emptyCredential.AddToCloneOptions(cloneOptions)
	require.Equal(t, expected, cloneOptions)
}
