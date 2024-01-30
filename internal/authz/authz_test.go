//
// Copyright 2024 Stacklok, Inc.
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

package authz_test

import (
	_ "embed"
	"encoding/json"
	"testing"

	fgasdk "github.com/openfga/go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/authz"
)

var (
	// Re-define authzModel variable since we don't want to export it
	//
	//go:embed model/minder.generated.json
	authzModel string
)

func TestAllRolesExistInFGAModel(t *testing.T) {
	t.Parallel()

	var m fgasdk.WriteAuthorizationModelRequest
	require.NoError(t, json.Unmarshal([]byte(authzModel), &m), "failed to unmarshal authz model")

	var projectTypeDef fgasdk.TypeDefinition
	var typedeffound bool
	for _, td := range m.TypeDefinitions {
		if td.Type == "project" {
			projectTypeDef = td
			typedeffound = true
			break
		}
	}

	require.True(t, typedeffound, "project type definition not found in authz model")

	t.Logf("relations: %v", projectTypeDef.Relations)

	for r := range authz.AllRoles {
		assert.Contains(t, *projectTypeDef.Relations, r.String(), "role %s not found in authz model", r)
	}
}
