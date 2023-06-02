//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
)

const (
	applicationID  int64 = 123456
	installationID int64 = 123456
)

func TestGenerateToken(t *testing.T) {
	a := New(applicationID, installationID, "../../test_files/private.pem")

	client, err := a.GitHubClient()
	if err != nil {
		panic(err)
	}

	// assert that client is a valid client
	assert.NotNil(t, client)
	assert.NotNil(t, client.Apps)
	assert.NotNil(t, client.Apps.Get)
	assert.NotNil(t, client.Apps.GetInstallation)
}
