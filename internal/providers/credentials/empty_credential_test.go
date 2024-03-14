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
