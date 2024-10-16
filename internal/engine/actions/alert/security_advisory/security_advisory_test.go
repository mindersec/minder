// Copyright 2024 Stacklok, Inc.
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
// Package rule provides the CLI subcommand for managing rules

// Package security_advisory provides necessary interfaces and implementations for
// creating alerts of type security advisory.

package security_advisory

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/db"
	enginerr "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/interfaces"
	"github.com/mindersec/minder/internal/profiles/models"
	mockghclient "github.com/mindersec/minder/internal/providers/github/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var TestActionTypeValid interfaces.ActionType = "alert-test"

func TestSecurityAdvisoryAlert(t *testing.T) {
	t.Parallel()

	saID := "123"

	tests := []struct {
		name             string
		actionType       interfaces.ActionType
		mockSetup        func(*mockghclient.MockGitHub)
		expectedErr      error
		expectedMetadata json.RawMessage
	}{
		{
			name:       "create a security advisory",
			actionType: TestActionTypeValid,
			mockSetup: func(mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					CreateSecurityAdvisory(gomock.Any(), gomock.Any(), gomock.Any(), pb.Severity_VALUE_HIGH.String(),
						gomock.Any(), gomock.Any(), gomock.Any()).
					Return(saID, nil)
			},
			expectedErr:      nil,
			expectedMetadata: json.RawMessage(fmt.Sprintf(`{"ghsa_id":"%s"}`, saID)),
		},
		{
			name:       "error from provider creating security advisory",
			actionType: TestActionTypeValid,
			mockSetup: func(mockGitHub *mockghclient.MockGitHub) {
				mockGitHub.EXPECT().
					CreateSecurityAdvisory(gomock.Any(), gomock.Any(), gomock.Any(), pb.Severity_VALUE_HIGH.String(),
						gomock.Any(), gomock.Any(), gomock.Any()).
					Return("", fmt.Errorf("failed to create security advisory"))
			},
			expectedErr:      enginerr.ErrActionFailed,
			expectedMetadata: json.RawMessage(nil),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() {
				ctrl.Finish()
			})

			ruleType := pb.RuleType{
				Name:                "rule_type_1",
				ShortFailureMessage: "This is a failure message",
				Def: &pb.RuleType_Definition{
					Alert:     &pb.RuleType_Definition_Alert{},
					Remediate: &pb.RuleType_Definition_Remediate{},
				},
			}
			saCfg := pb.RuleType_Definition_Alert_AlertTypeSA{
				Severity: pb.Severity_VALUE_HIGH.String(),
			}

			mockClient := mockghclient.NewMockGitHub(ctrl)
			tt.mockSetup(mockClient)

			saAlert, err := NewSecurityAdvisoryAlert(tt.actionType, &ruleType, &saCfg, mockClient)
			require.NoError(t, err)
			require.NotNil(t, saAlert)

			evalParams := &interfaces.EvalStatusParams{
				EvalStatusFromDb: &db.ListRuleEvaluationsByProfileIdRow{},
				Profile:          &models.ProfileAggregate{},
				Rule:             &models.RuleInstance{},
			}

			retMeta, err := saAlert.Do(
				context.Background(),
				interfaces.ActionCmdOn,
				models.ActionOptOn,
				&pb.PullRequest{},
				evalParams,
				nil,
			)
			require.ErrorIs(t, err, tt.expectedErr, "expected error")
			require.Equal(t, tt.expectedMetadata, retMeta)
		})
	}
}
