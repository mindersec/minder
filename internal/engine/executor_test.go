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
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/structpb"

	mockdb "github.com/stacklok/minder/database/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/engine/actions/alert"
	"github.com/stacklok/minder/internal/engine/actions/remediate"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/util/testqueue"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
	projectID := uuid.New()
	providerName := "github"
	providerID := uuid.New()
	passthroughRuleType := "passthrough"
	profileID := uuid.New()
	ruleTypeID := uuid.New()
	repositoryID := uuid.New()
	executionID := uuid.New()

	authtoken := generateFakeAccessToken(t)

	// -- start expectations

	// not valuable yet, but would have to be updated once actions start using this
	mockStore.EXPECT().GetRuleEvaluationByProfileIdAndRuleType(gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(db.ListRuleEvaluationsByProfileIdRow{}, nil)

	// get project information
	mockStore.EXPECT().
		GetParentProjects(gomock.Any(), projectID).
		Return([]uuid.UUID{
			projectID,
		}, nil)

	mockStore.EXPECT().
		GetProviderByName(gomock.Any(), db.GetProviderByNameParams{
			Name: providerName,
			Projects: []uuid.UUID{
				projectID,
			},
		}).
		Return(db.Provider{
			ID:        providerID,
			Name:      providerName,
			ProjectID: projectID,
		}, nil)

	// get access token
	mockStore.EXPECT().
		GetAccessTokenByProjectID(gomock.Any(),
			db.GetAccessTokenByProjectIDParams{
				Provider:  providerName,
				ProjectID: projectID,
			}).
		Return(db.ProviderAccessToken{
			EncryptedToken: authtoken,
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
		ListProfilesByProjectID(gomock.Any(), projectID).
		Return([]db.ListProfilesByProjectIDRow{
			{
				ID:              profileID,
				Name:            "test-profile",
				Entity:          db.EntitiesRepository,
				Provider:        providerName,
				ProjectID:       projectID,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				ContextualRules: json.RawMessage(marshalledCRS),
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
		GetRuleTypeByName(gomock.Any(), db.GetRuleTypeByNameParams{
			Provider:  providerName,
			ProjectID: projectID,
			Name:      passthroughRuleType,
		}).Return(db.RuleType{
		ID:         ruleTypeID,
		Name:       passthroughRuleType,
		Provider:   providerName,
		ProjectID:  projectID,
		Definition: json.RawMessage(marshalledRTD),
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

	// Mock upserting remediate status
	ruleEvalRemediationId := uuid.New()
	mockStore.EXPECT().
		UpsertRuleDetailsRemediate(gomock.Any(), db.UpsertRuleDetailsRemediateParams{
			RuleEvalID: ruleEvalId,
			Status:     db.RemediationStatusTypesSkipped,
			Details:    "",
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
			Entity:        db.EntitiesRepository,
			RepositoryID:  repositoryID,
			ArtifactID:    uuid.NullUUID{},
			PullRequestID: uuid.NullUUID{},
			LockedBy:      executionID,
		}).Return(nil)

	// Mock release lock
	mockStore.EXPECT().
		ReleaseLock(gomock.Any(), db.ReleaseLockParams{
			Entity:        db.EntitiesRepository,
			RepositoryID:  repositoryID,
			ArtifactID:    uuid.NullUUID{},
			PullRequestID: uuid.NullUUID{},
			LockedBy:      executionID,
		}).Return(nil)

	// -- end expectations

	tmpdir := t.TempDir()
	// write token key to file
	tokenKeyPath := tmpdir + "/token_key"

	// write key to file
	err = os.WriteFile(tokenKeyPath, []byte(fakeTokenKey), 0600)
	require.NoError(t, err, "expected no error")

	evt, err := events.Setup(context.Background(), &serverconfig.EventConfig{
		Driver: "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{
			BlockPublishUntilSubscriberAck: true,
		},
	})
	require.NoError(t, err, "failed to setup eventer")

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	go func() {
		t.Log("Running eventer")
		evt.Register(events.FlushEntityEventTopic, pq.Pass)
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	testTimeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	e, err := engine.NewExecutor(ctx, mockStore, &serverconfig.AuthConfig{
		TokenKey: tokenKeyPath,
	}, evt)
	require.NoError(t, err, "expected no error")

	eiw := entities.NewEntityInfoWrapper().
		WithProvider(providerName).
		WithProjectID(projectID).
		WithRepository(&minderv1.Repository{
			Name:     "test",
			RepoId:   123,
			CloneUrl: "github.com/foo/bar.git",
		}).WithRepositoryID(repositoryID).
		WithExecutionID(executionID)

	t.Log("waiting for eventer to start")
	<-evt.Running()

	msg, err := eiw.BuildMessage()
	require.NoError(t, err, "expected no error")

	ts := &logger.TelemetryStore{
		Project:    projectID,
		Provider:   providerName,
		Repository: repositoryID,
	}
	ctx = ts.WithTelemetry(ctx)
	msg.SetContext(ctx)

	// Run in the background
	go func() {
		t.Log("Running entity event handler")
		require.NoError(t, e.HandleEntityEvent(msg), "expected no error")
	}()

	// expect flush
	t.Log("waiting for flush")
	require.NotNil(t, <-queued, "expected message")

	require.NoError(t, evt.Close(), "expected no error")

	t.Log("waiting for executor to finish")
	e.Wait()

	require.Len(t, ts.Evals, 1, "expected one eval to be logged")
	requredEval := ts.Evals[0]
	require.Equal(t, "test-profile", requredEval.Profile.Name)
	require.Equal(t, "success", requredEval.EvalResult)
	require.Equal(t, "passthrough", requredEval.RuleType.Name)
	require.Equal(t, "off", requredEval.Actions[alert.ActionType].State)
	require.Equal(t, "off", requredEval.Actions[remediate.ActionType].State)
}
