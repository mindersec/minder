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

package controlplane

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/engine"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ListEvaluationResults lists the evaluation results for entities filtered by entity type, labels, profiles, and rule types.
func (s *Server) ListEvaluationResults(
	ctx context.Context,
	_ *minderv1.ListEvaluationResultsRequest,
) (*minderv1.ListEvaluationResultsResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)

	err := entityCtx.Validate(ctx, s.store)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error in entity context: %v", err)
	}

	// TODO: Implement the ListEvaluationResults handler
	// Until then, return a mocked response
	return mockListEvaluationResults(), nil
}

func mockListEvaluationResults() *minderv1.ListEvaluationResultsResponse {
	res := []*minderv1.ListEvaluationResultsResponse_EntityEvaluationResults{}
	res = append(res, &minderv1.ListEvaluationResultsResponse_EntityEvaluationResults{
		Entity: &minderv1.EntityTypedId{
			Type: minderv1.Entity_ENTITY_REPOSITORIES,
			Id:   "stacklok/depot",
		},
		Profiles: []*minderv1.ListEvaluationResultsResponse_EntityProfileEvaluationResults{
			{
				Profile: "my-profile",
				Results: []*minderv1.RuleEvaluationStatus{
					{
						ProfileId:   "a4d5589b-cf5a-42ab-a940-15d2a8c5a3e1",
						RuleId:      "12d4c5e8-7a6e-4a50-afba-8883d9a45673",
						Entity:      "repository",
						Status:      "failure",
						LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix()},
						EntityInfo: map[string]string{
							"provider":      "github",
							"repo_name":     "depot",
							"repo_owner":    "stacklok",
							"repository_id": "a2b6392d-3b4f-48a9-b641-dc9c595c3daf",
						},
						Details:                "Security alerts are enabled for the repository.",
						Guidance:               "Keep the security alerts feature enabled to stay informed about potential vulnerabilities.",
						RemediationStatus:      "not available",
						RemediationLastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 120},
						RuleTypeName:           "branch_protection_lock_branch",
						RuleDescriptionName:    "branch_protection_lock_branch",
						AlertStatus:            "on",
						AlertLastUpdated:       &timestamppb.Timestamp{Seconds: time.Now().Unix() - 100},
						AlertDetails:           "",
						AlertMetadata:          map[string]string{"ghsa_id": "GHSA-1234-5678"},
					},
					{
						ProfileId:   "a4d5589b-cf5a-42ab-a940-15d2a8c5a3e1",
						RuleId:      "12d4c5e8-7a6e-4a50-afba-8883d9a45673",
						Entity:      "repository",
						Status:      "success",
						LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix()},
						EntityInfo: map[string]string{
							"provider":      "github",
							"repo_name":     "depot",
							"repo_owner":    "stacklok",
							"repository_id": "a2b6392d-3b4f-48a9-b641-dc9c595c3daf",
						},
						Details:                "Security alerts are enabled for the repository.",
						Guidance:               "Keep the security alerts feature enabled to stay informed about potential vulnerabilities.",
						RemediationStatus:      "not available",
						RemediationLastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 120},
						RuleTypeName:           "branch_protection_lock_branch",
						RuleDescriptionName:    "branch_protection_lock_branch",
						AlertStatus:            "off",
						AlertLastUpdated:       &timestamppb.Timestamp{Seconds: time.Now().Unix() - 100},
						AlertDetails:           "",
						AlertMetadata:          map[string]string{},
					},
				},
			},
		},
	},
	)
	res = append(res, &minderv1.ListEvaluationResultsResponse_EntityEvaluationResults{
		Entity: &minderv1.EntityTypedId{
			Type: minderv1.Entity_ENTITY_REPOSITORIES,
			Id:   "stacklok/insight",
		},
		Profiles: []*minderv1.ListEvaluationResultsResponse_EntityProfileEvaluationResults{
			{
				Profile: "my-profile-2",
				Results: []*minderv1.RuleEvaluationStatus{
					{
						ProfileId:   "ebd3f978-f2b0-4cd9-a9de-2536a2414a37",
						RuleId:      "9984ecef-b8b9-41d4-8d02-748889d3aef2",
						Entity:      "repository",
						Status:      "failure",
						LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 100},
						EntityInfo: map[string]string{
							"provider":      "github",
							"repo_name":     "insight",
							"repo_owner":    "stacklok",
							"repository_id": "9d3ef47a-3d6f-43f8-9a29-d9112fbc92f3",
						},
						Details:                "denied",
						Guidance:               "Configure branch protection rules to require code reviews before merging.",
						RemediationStatus:      "pending",
						RemediationLastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 200},
						RuleTypeName:           "artifact_signature",
						RuleDescriptionName:    "artifact_signature",
						AlertStatus:            "off",
						AlertLastUpdated:       &timestamppb.Timestamp{Seconds: time.Now().Unix() - 180},
						AlertDetails:           "",
						AlertMetadata:          map[string]string{},
					},
				},
			},
		},
	},
	)
	res = append(res, &minderv1.ListEvaluationResultsResponse_EntityEvaluationResults{
		Entity: &minderv1.EntityTypedId{
			Type: minderv1.Entity_ENTITY_REPOSITORIES,
			Id:   "stacklok/streamline",
		},
		Profiles: []*minderv1.ListEvaluationResultsResponse_EntityProfileEvaluationResults{
			{
				Profile: "my-profile-2",
				Results: []*minderv1.RuleEvaluationStatus{
					{
						ProfileId:   "c5bd9d1b-3d96-4a57-a5a0-3d6248c3b677",
						RuleId:      "2aaf7f8c-76c2-499e-b667-9a4d28b3c295",
						Entity:      "repository",
						Status:      "error",
						LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 30},
						EntityInfo: map[string]string{
							"provider":      "github",
							"repo_name":     "streamline",
							"repo_owner":    "stacklok",
							"repository_id": "b3229c1e-6f2c-4b4b-9f58-5b690f2af6f9",
						},
						Details:                "evaluation skipped: rule not applicable",
						Guidance:               "Update dependencies to their latest versions to improve security and compatibility.",
						RemediationStatus:      "skipped",
						RemediationLastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 150},
						RuleTypeName:           "dependabot_configured",
						RuleDescriptionName:    "dependabot_configured",
						AlertStatus:            "error",
						AlertLastUpdated:       &timestamppb.Timestamp{Seconds: time.Now().Unix() - 120},
						AlertDetails:           "",
						AlertMetadata:          map[string]string{},
					},
					{
						ProfileId:   "c5bd9d1b-3d96-4a57-a5a0-3d6248c3b677",
						RuleId:      "2aaf7f8c-76c2-499e-b667-9a4d28b3c295",
						Entity:      "repository",
						Status:      "failure",
						LastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 30},
						EntityInfo: map[string]string{
							"provider":      "github",
							"repo_name":     "streamline",
							"repo_owner":    "stacklok",
							"repository_id": "b3229c1e-6f2c-4b4b-9f58-5b690f2af6f9",
						},
						Details:                "denied",
						Guidance:               "Update dependencies to their latest versions to improve security and compatibility.",
						RemediationStatus:      "skipped",
						RemediationLastUpdated: &timestamppb.Timestamp{Seconds: time.Now().Unix() - 150},
						RuleTypeName:           "automatic_branch_deletion",
						RuleDescriptionName:    "automatic_branch_deletion",
						AlertStatus:            "on",
						AlertLastUpdated:       &timestamppb.Timestamp{Seconds: time.Now().Unix() - 120},
						AlertDetails:           "",
						AlertMetadata:          map[string]string{"ghsa_id": "GHSA-1234-5678"},
					},
				},
			},
		},
	},
	)
	return &minderv1.ListEvaluationResultsResponse{
		Entities: res,
	}
}
