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

	"github.com/stretchr/testify/require"
)

// A helper function to create a random policy
func createRandomPolicy(t *testing.T, group int32) Policy {
	arg := CreatePolicyParams{
		Provider:         "github",
		GroupID:          group,
		PolicyType:       PolicyTypePOLICYTYPEBRANCHPROTECTION,
		PolicyDefinition: json.RawMessage(""),
	}

	policy, err := testQueries.CreatePolicy(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, policy)

	require.Equal(t, arg.GroupID, policy.GroupID)
	require.Equal(t, arg.Provider, policy.Provider)
	require.Equal(t, arg.PolicyType, policy.PolicyType)
	require.Equal(t, policy.PolicyDefinition, "key: value\n")
	require.NotZero(t, policy.CreatedAt)
	require.NotZero(t, policy.UpdatedAt)

	return policy
}

func TestPolicy(t *testing.T) {
	org := createRandomOrganization(t)
	group := createRandomGroup(t, org.ID)
	createRandomPolicy(t, group.ID)
}

func TestGetPolicy(t *testing.T) {
	org := createRandomOrganization(t)
	group := createRandomGroup(t, org.ID)
	policy1 := createRandomPolicy(t, group.ID)

	policy2, err := testQueries.GetPolicyByID(context.Background(), policy1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, policy2)

	require.Equal(t, policy1.ID, policy2.ID)
	require.Equal(t, policy1.GroupID, policy2.GroupID)
	require.Equal(t, policy1.Provider, policy2.Provider)
	require.Equal(t, policy1.PolicyType, policy2.PolicyType)
	require.Equal(t, policy1.PolicyDefinition, policy2.PolicyDefinition)
	require.NotZero(t, policy2.CreatedAt)
	require.NotZero(t, policy2.UpdatedAt)
}

func TestDeletePolicy(t *testing.T) {
	org := createRandomOrganization(t)
	group := createRandomGroup(t, org.ID)
	policy := createRandomPolicy(t, group.ID)

	err := testQueries.DeletePolicy(context.Background(), policy.ID)
	require.NoError(t, err)

	policy2, err := testQueries.GetPolicyByID(context.Background(), policy.ID)

	require.Error(t, err)
	require.Empty(t, policy2)
}
