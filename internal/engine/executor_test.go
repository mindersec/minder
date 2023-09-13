// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine_test

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/structpb"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/internal/crypto"
	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/engine"
	"github.com/stacklok/mediator/internal/entities"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

const (
	fakeTokenKey = "foo-bar"
)

func generateFakeAccessToken(t *testing.T) string {
	t.Helper()

	ftoken := &oauth2.Token{
		AccessToken:  "foo-bar",
		TokenType:    "bar-baz",
		RefreshToken: "",
		// Expires in 10 mins
		Expiry: time.Now().Add(10 * time.Minute),
	}

	// Convert token to JSON
	jsonData, err := json.Marshal(ftoken)
	require.NoError(t, err, "expected no error")

	// encode token
	encryptedToken, err := crypto.EncryptBytes(fakeTokenKey, jsonData)
	require.NoError(t, err, "expected no error")

	return base64.StdEncoding.EncodeToString(encryptedToken)
}

func TestExecutor_handleEntityEvent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	// declarations
	groupID := int32(1)
	provider := "github"
	passthroughRuleType := "passthrough"

	authtoken := generateFakeAccessToken(t)

	// -- start expectations

	// get group information
	mockStore.EXPECT().
		GetGroupByID(gomock.Any(), groupID).
		Return(db.Group{
			ID:   groupID,
			Name: "test",
		}, nil)

	// get access token
	mockStore.EXPECT().
		GetAccessTokenByGroupID(gomock.Any(),
			db.GetAccessTokenByGroupIDParams{
				Provider: provider,
				GroupID:  groupID,
			}).
		Return(db.ProviderAccessToken{
			EncryptedToken: authtoken,
		}, nil)

	// list one policy
	crs := []*pb.PipelinePolicy_ContextualRuleSet{
		{
			Context: &provider,
			Rules: []*pb.PipelinePolicy_Rule{
				{
					Type: passthroughRuleType,
					Def:  &structpb.Struct{},
				},
			},
		},
	}

	marshalledCRS, err := json.Marshal(crs)
	require.NoError(t, err, "expected no error")

	mockStore.EXPECT().
		ListPoliciesByGroupID(gomock.Any(), groupID).
		Return([]db.ListPoliciesByGroupIDRow{
			{
				ID:              1,
				Name:            "test-policy",
				Entity:          db.EntitiesRepository,
				Provider:        provider,
				GroupID:         groupID,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				ContextualRules: json.RawMessage(marshalledCRS),
			},
		}, nil)

	// get relevant rule
	ruleTypeDef := &pb.RuleType_Definition{
		InEntity:   entities.RepositoryEntity.String(),
		RuleSchema: &structpb.Struct{},
		Ingest: &pb.RuleType_Definition_Ingest{
			Type: "builtin",
			Builtin: &pb.BuiltinType{
				Method: "Passthrough",
			},
		},
		Eval: &pb.RuleType_Definition_Eval{
			Type: "rego",
			Rego: &pb.RuleType_Definition_Eval_Rego{
				Type: "deny-by-default",
				Def: `package mediator
default allow = true`,
			},
		},
	}

	marshalledRTD, err := json.Marshal(ruleTypeDef)
	require.NoError(t, err, "expected no error")

	mockStore.EXPECT().
		GetRuleTypeByName(gomock.Any(), db.GetRuleTypeByNameParams{
			Provider: provider,
			GroupID:  groupID,
			Name:     passthroughRuleType,
		}).Return(db.RuleType{
		ID:         1,
		Name:       passthroughRuleType,
		Provider:   provider,
		GroupID:    groupID,
		Definition: json.RawMessage(marshalledRTD),
	}, nil)

	// Upload passing status
	mockStore.EXPECT().
		UpsertRuleEvaluationStatus(gomock.Any(), db.UpsertRuleEvaluationStatusParams{
			PolicyID: 1,
			RepositoryID: sql.NullInt32{
				Int32: 123,
				Valid: true,
			},
			ArtifactID: sql.NullInt32{},
			RuleTypeID: 1,
			Entity:     db.EntitiesRepository,
			EvalStatus: db.EvalStatusTypesSuccess,
		}).Return(nil)

	// -- end expectations

	tmpdir := t.TempDir()
	// write token key to file
	tokenKeyPath := tmpdir + "/token_key"

	// write key to file
	err = os.WriteFile(tokenKeyPath, []byte(fakeTokenKey), 0600)
	require.NoError(t, err, "expected no error")

	e, err := engine.NewExecutor(mockStore, &config.AuthConfig{
		TokenKey: tokenKeyPath,
	})
	require.NoError(t, err, "expected no error")

	eiw := engine.NewEntityInfoWrapper().
		WithProvider(provider).
		WithGroupID(groupID).
		WithRepository(&pb.RepositoryResult{
			Repository: "test",
			RepoId:     123,
			CloneUrl:   "github.com/foo/bar.git",
		}).WithRepositoryID(123)

	msg, err := eiw.BuildMessage()
	require.NoError(t, err, "expected no error")

	require.NoError(t, e.HandleEntityEvent(msg), "expected no error")
}
