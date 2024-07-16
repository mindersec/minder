// Copyright 2023 Stacklok, Inc.
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

package engine_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/structpb"

	mockdb "github.com/stacklok/minder/database/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/engine/actions/alert"
	"github.com/stacklok/minder/internal/engine/actions/remediate"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/flags"
	mockhistory "github.com/stacklok/minder/internal/history/mock"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/metrics/meters"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/github/clients"
	ghmanager "github.com/stacklok/minder/internal/providers/github/manager"
	ghService "github.com/stacklok/minder/internal/providers/github/service"
	"github.com/stacklok/minder/internal/providers/manager"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func TestExecutor_handleEntityEvent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	// declarations
	projectID := uuid.New()
	providerName := "github"
	providerID := uuid.New()
	passthroughRuleType := "passthrough"
	profileID := uuid.New()
	ruleTypeID := uuid.New()
	repositoryID := uuid.New()
	executionID := uuid.New()

	// write token key to file
	tmpdir := t.TempDir()
	tokenKeyPath := tmpdir + "/token_key"

	// generate 256-bit key
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	require.NoError(t, err)
	encodedKey := base64.StdEncoding.EncodeToString(key)

	// write key to file
	err = os.WriteFile(tokenKeyPath, []byte(encodedKey), 0600)
	require.NoError(t, err, "expected no error")

	// Needed to keep these tests working as-is.
	// In future, beef up unit test coverage in the dependencies
	// of this code, and refactor these tests to use stubs.
	config := &serverconfig.Config{
		Auth: serverconfig.AuthConfig{TokenKey: tokenKeyPath},
	}
	cryptoEngine, err := crypto.NewEngineFromConfig(config)
	require.NoError(t, err)

	authtoken := generateFakeAccessToken(t, cryptoEngine)
	// -- start expectations

	// not valuable yet, but would have to be updated once actions start using this
	mockStore.EXPECT().GetRuleEvaluationByProfileIdAndRuleType(gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(db.ListRuleEvaluationsByProfileIdRow{}, nil)

	mockStore.EXPECT().
		GetProviderByID(gomock.Any(), gomock.Eq(providerID)).
		Return(db.Provider{
			ID:        providerID,
			Name:      providerName,
			ProjectID: projectID,
			Class:     db.ProviderClassGithub,
			Version:   provinfv1.V1,
			Implements: []db.ProviderType{
				db.ProviderTypeGithub,
			},
			Definition: json.RawMessage(`{"github": {}}`),
		}, nil)

	// get access token
	mockStore.EXPECT().
		GetAccessTokenByProjectID(gomock.Any(),
			db.GetAccessTokenByProjectIDParams{
				Provider:  providerName,
				ProjectID: projectID,
			}).
		Return(db.ProviderAccessToken{
			EncryptedAccessToken: authtoken,
		}, nil)

	// list one profile
	crs := []*minderv1.Profile_Rule{
		{
			Type: passthroughRuleType,
			Name: passthroughRuleType,
			Def:  &structpb.Struct{},
		},
	}

	marshalledCRS, err := json.Marshal(crs)
	require.NoError(t, err, "expected no error")

	mockStore.EXPECT().
		GetParentProjects(gomock.Any(), projectID).
		Return([]uuid.UUID{projectID}, nil).
		Times(2)

	mockStore.EXPECT().
		ListProfilesByProjectID(gomock.Any(), projectID).
		Return([]db.ListProfilesByProjectIDRow{
			{
				Profile: db.Profile{
					ID:        profileID,
					Name:      "test-profile",
					ProjectID: projectID,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				ProfilesWithEntityProfile: db.ProfilesWithEntityProfile{
					Entity: db.NullEntities{
						Entities: db.EntitiesRepository,
						Valid:    true,
					},
					ContextualRules: pqtype.NullRawMessage{
						RawMessage: marshalledCRS,
						Valid:      true,
					},
				},
			},
		}, nil)

	// get relevant rule
	ruleTypeDef := &minderv1.RuleType_Definition{
		InEntity:   minderv1.RepositoryEntity.String(),
		RuleSchema: &structpb.Struct{},
		Ingest: &minderv1.RuleType_Definition_Ingest{
			Type: "builtin",
			Builtin: &minderv1.BuiltinType{
				Method: "Passthrough",
			},
		},
		Eval: &minderv1.RuleType_Definition_Eval{
			Type: "rego",
			Rego: &minderv1.RuleType_Definition_Eval_Rego{
				Type: "deny-by-default",
				Def: `package minder
default allow = true`,
			},
		},
	}

	marshalledRTD, err := json.Marshal(ruleTypeDef)
	require.NoError(t, err, "expected no error")

	mockStore.EXPECT().
		GetRuleTypeIDByRuleNameEntityProfile(gomock.Any(), gomock.Any()).
		Return(ruleTypeID, nil)

	mockStore.EXPECT().
		GetRuleTypesByEntityInHierarchy(gomock.Any(), db.GetRuleTypesByEntityInHierarchyParams{
			EntityType: db.EntitiesRepository,
			Projects:   []uuid.UUID{projectID},
		}).
		Return([]db.RuleType{
			{
				ID:         ruleTypeID,
				Name:       passthroughRuleType,
				ProjectID:  projectID,
				Definition: marshalledRTD,
			},
		}, nil)

	ruleEvalId := uuid.New()

	// Upload passing status
	mockStore.EXPECT().
		UpsertRuleEvaluations(gomock.Any(), db.UpsertRuleEvaluationsParams{
			ProfileID: profileID,
			RepositoryID: uuid.NullUUID{
				UUID:  repositoryID,
				Valid: true,
			},
			ArtifactID: uuid.NullUUID{},
			RuleTypeID: ruleTypeID,
			Entity:     db.EntitiesRepository,
			RuleName:   passthroughRuleType,
		}).Return(ruleEvalId, nil)

	// Mock upserting eval details status
	ruleEvalDetailsId := uuid.New()
	mockStore.EXPECT().
		UpsertRuleDetailsEval(gomock.Any(), db.UpsertRuleDetailsEvalParams{
			RuleEvalID: ruleEvalId,
			Status:     db.EvalStatusTypesSuccess,
			Details:    "",
		}).Return(ruleEvalDetailsId, nil)

	ruleInstanceID := uuid.New()
	mockStore.EXPECT().
		GetIDByProfileEntityName(gomock.Any(), db.GetIDByProfileEntityNameParams{
			ProfileID:  profileID,
			EntityType: db.EntitiesRepository,
			Name:       passthroughRuleType,
		}).
		Return(ruleInstanceID, nil)

	// Mock upserting remediate status
	ruleEvalRemediationId := uuid.New()
	mockStore.EXPECT().
		UpsertRuleDetailsRemediate(gomock.Any(), db.UpsertRuleDetailsRemediateParams{
			RuleEvalID: ruleEvalId,
			Status:     db.RemediationStatusTypesSkipped,
			Details:    "",
			Metadata:   json.RawMessage("{}"),
		}).Return(ruleEvalRemediationId, nil)
	// Empty metadata
	meta, _ := json.Marshal(map[string]any{})
	// Mock upserting alert status
	ruleEvalAlertId := uuid.New()
	mockStore.EXPECT().
		UpsertRuleDetailsAlert(gomock.Any(), db.UpsertRuleDetailsAlertParams{
			RuleEvalID: ruleEvalId,
			Status:     db.AlertStatusTypesSkipped,
			Metadata:   meta,
			Details:    "",
		}).Return(ruleEvalAlertId, nil)

	// Mock update lease for lock
	mockStore.EXPECT().
		UpdateLease(gomock.Any(), db.UpdateLeaseParams{
			Entity: db.EntitiesRepository,
			RepositoryID: uuid.NullUUID{
				UUID:  repositoryID,
				Valid: true,
			},
			ArtifactID:    uuid.NullUUID{},
			PullRequestID: uuid.NullUUID{},
			LockedBy:      executionID,
		}).Return(nil)

	// Mock release lock
	mockStore.EXPECT().
		ReleaseLock(gomock.Any(), db.ReleaseLockParams{
			Entity:        db.EntitiesRepository,
			RepositoryID:  uuid.NullUUID{UUID: repositoryID, Valid: true},
			ArtifactID:    uuid.NullUUID{},
			PullRequestID: uuid.NullUUID{},
			LockedBy:      executionID,
		}).Return(nil)

	// -- end expectations

	ghProviderService := ghService.NewGithubProviderService(
		mockStore,
		cryptoEngine,
		metrics.NewNoopMetrics(),
		// These nil dependencies do not matter for the current tests
		nil,
		nil,
		clients.NewGitHubClientFactory(telemetry.NewNoopMetrics()),
	)

	githubProviderManager := ghmanager.NewGitHubProviderClassManager(
		&ratecache.NoopRestClientCache{},
		clients.NewGitHubClientFactory(telemetry.NewNoopMetrics()),
		&serverconfig.ProviderConfig{},
		nil,
		cryptoEngine,
		nil,
		mockStore,
		ghProviderService,
	)

	providerStore := providers.NewProviderStore(mockStore)
	providerManager, err := manager.NewProviderManager(providerStore, githubProviderManager)
	require.NoError(t, err)

	execMetrics, err := engine.NewExecutorMetrics(&meters.NoopMeterFactory{})
	require.NoError(t, err)
	historyService := mockhistory.NewMockEvaluationHistoryService(ctrl)

	executor := engine.NewExecutor(
		mockStore,
		providerManager,
		execMetrics,
		historyService,
		&flags.FakeClient{},
	)

	eiw := entities.NewEntityInfoWrapper().
		WithProviderID(providerID).
		WithProjectID(projectID).
		WithRepository(&minderv1.Repository{
			Name:     "test",
			RepoId:   123,
			CloneUrl: "github.com/foo/bar.git",
		}).WithRepositoryID(repositoryID).
		WithExecutionID(executionID)

	ts := &logger.TelemetryStore{
		Project:    projectID,
		ProviderID: providerID,
		Repository: repositoryID,
	}
	ctx := ts.WithTelemetry(context.Background())

	err = executor.EvalEntityEvent(ctx, eiw)
	require.NoError(t, err)

	require.Len(t, ts.Evals, 1, "expected one eval to be logged")
	requredEval := ts.Evals[0]
	require.Equal(t, "test-profile", requredEval.Profile.Name)
	require.Equal(t, "success", requredEval.EvalResult)
	require.Equal(t, "passthrough", requredEval.RuleType.Name)
	require.Equal(t, "off", requredEval.Actions[alert.ActionType].State)
	require.Equal(t, "off", requredEval.Actions[remediate.ActionType].State)
}

func generateFakeAccessToken(t *testing.T, cryptoEngine crypto.Engine) pqtype.NullRawMessage {
	t.Helper()

	ftoken := &oauth2.Token{
		AccessToken:  "foo-bar",
		TokenType:    "bar-baz",
		RefreshToken: "",
		// Expires in 10 mins
		Expiry: time.Now().Add(10 * time.Minute),
	}

	// encrypt token
	encryptedToken, err := cryptoEngine.EncryptOAuthToken(ftoken)
	require.NoError(t, err)
	serialized, err := encryptedToken.Serialize()
	require.NoError(t, err)
	return pqtype.NullRawMessage{
		RawMessage: serialized,
		Valid:      true,
	}
}
