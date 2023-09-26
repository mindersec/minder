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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package db

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/mediator/internal/util"
)

func createRandomProvider(t *testing.T, groupID int32) Provider {
	t.Helper()

	seed := time.Now().UnixNano()

	prov, err := testQueries.CreateProvider(context.Background(), CreateProviderParams{
		Name:       util.RandomName(seed),
		GroupID:    groupID,
		Implements: []ProviderType{ProviderTypeGithub, ProviderTypeGit},
		Definition: json.RawMessage("{}"),
	})
	require.NoError(t, err, "Error creating provider")
	require.NotEmpty(t, prov, "Empty provider returned")

	return prov
}
