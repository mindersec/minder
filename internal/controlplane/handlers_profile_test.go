// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/db/embedded"
	"github.com/mindersec/minder/internal/engine/engcontext"
	stubeventer "github.com/mindersec/minder/internal/events/stubs"
	"github.com/mindersec/minder/internal/providers"
	"github.com/mindersec/minder/internal/util"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/selectors"
	"github.com/mindersec/minder/pkg/profiles"
)

//nolint:gocyclo
func TestCreateProfile(t *testing.T) {
	t.Parallel()

	// We can't use mockdb.NewMockStore because BeginTransaction returns a *sql.Tx,
	// which is not an interface and can't be mocked.
	dbStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	if err != nil {
		t.Fatalf("Error creating fake store: %v", err)
	}

	// Common database setup
	ctx := context.Background()
	dbproj, err := dbStore.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Error creating project: %v", err)
	}

	_, err = dbStore.CreateRuleType(ctx, db.CreateRuleTypeParams{
		Name:          "rule_type_1",
		ProjectID:     dbproj.ID,
		Definition:    []byte(`{"in_entity": "repository","ruleSchema":{}}`),
		SeverityValue: db.SeverityLow,
		ReleasePhase:  db.ReleaseStatusAlpha,
	})
	if err != nil {
		t.Fatalf("Error creating rule type: %v", err)
	}

	tests := []struct {
		name    string
		profile *minderv1.CreateProfileRequest
		wantErr string
		result  *minderv1.CreateProfileResponse
	}{{
		name: "Create profile with empty name",
		profile: &minderv1.CreateProfileRequest{
			Profile: &minderv1.Profile{
				Name: "",
			},
		},
		wantErr: `Couldn't create profile: validation failed: profile name cannot be empty`,
	}, {
		name: "Create profile with invalid name",
		profile: &minderv1.CreateProfileRequest{
			Profile: &minderv1.Profile{
				Name: "colon:invalid",
			},
		},
		wantErr: `Couldn't create profile: validation failed: name may only contain letters, numbers, hyphens and underscores, and is limited to a maximum of 63 characters`,
	}, {
		name: "Create profile with no rules",
		profile: &minderv1.CreateProfileRequest{
			Profile: &minderv1.Profile{
				Name: "test_norules",
			},
		},
		result: &minderv1.CreateProfileResponse{
			Profile: &minderv1.Profile{
				Name:      "test_norules",
				Alert:     proto.String("on"),
				Remediate: proto.String("off"),
			},
		},
	},
		{
			name: "Create profile with valid name and rules",
			profile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test",
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
			result: &minderv1.CreateProfileResponse{
				Profile: &minderv1.Profile{
					Name:      "test",
					Alert:     proto.String("on"),
					Remediate: proto.String("off"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Name: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
		},
		{
			name: "Create profile with explicit alert and remediate",
			profile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:      "test_explicit",
					Alert:     proto.String("off"),
					Remediate: proto.String("on"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
			result: &minderv1.CreateProfileResponse{
				Profile: &minderv1.Profile{
					Name:      "test_explicit",
					Alert:     proto.String("off"),
					Remediate: proto.String("on"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Name: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
		},
		{
			name: "Create profile with explicit display name",
			profile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:        "test_explicit_display",
					DisplayName: "This is an explicit display name",
					Alert:       proto.String("off"),
					Remediate:   proto.String("on"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
			result: &minderv1.CreateProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_explicit_display",
					DisplayName: "This is an explicit display name",
					Alert:       proto.String("off"),
					Remediate:   proto.String("on"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Name: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
				},
			},
		},
		{
			name: "Profile with selectors",
			profile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:      "test_selectors",
					Alert:     proto.String("off"),
					Remediate: proto.String("off"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "repository",
							Selector:    "repository.name != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
					},
				},
			},
			result: &minderv1.CreateProfileResponse{
				Profile: &minderv1.Profile{
					Name:      "test_selectors",
					Alert:     proto.String("off"),
					Remediate: proto.String("off"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Name: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "repository",
							Selector:    "repository.name != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
					},
				},
			},
		},
		{
			name: "Selector with wrong entity",
			profile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:      "test_selectors_bad_entity",
					Alert:     proto.String("off"),
					Remediate: proto.String("off"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "no_such_entity",
							Selector:    "repository.name != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
					},
				},
			},
			wantErr: `unsupported entity type: invalid entity type no_such_entity: unsupported entity type`,
		},
		{
			name: "Selector does not parse",
			profile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:      "test_selectors_syntax_error",
					Alert:     proto.String("off"),
					Remediate: proto.String("off"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "repository",
							Selector:    "repository.name != ",
							Description: "Exclude stacklok/demo-repo-go",
						},
					},
				},
			},
			wantErr: `selector failed to parse: Syntax error: mismatched input '<EOF>' expecting {'[', '{', '(', '.', '-', '!', 'true', 'false', 'null', NUM_FLOAT, NUM_INT, NUM_UINT, STRING, BYTES, IDENTIFIER}`,
		},
		{
			name: "Selector uses wrong attribute",
			profile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:      "test_selectors_attr_error",
					Alert:     proto.String("off"),
					Remediate: proto.String("off"),
					Repository: []*minderv1.Profile_Rule{{
						Type: "rule_type_1",
						Def:  &structpb.Struct{},
					}},
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "repository",
							Selector:    "repository.foo != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
					},
				},
			},
			wantErr: `selector is invalid: undefined field 'foo'`,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.profile.GetContext() == nil {
				tc.profile.Profile.Context = &minderv1.Context{
					Project: proto.String(dbproj.ID.String()),
				}
				if tc.result != nil {
					tc.result.GetProfile().Context = tc.profile.GetContext()
				}
			}

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: dbproj.ID},
			})

			evts := &stubeventer.StubEventer{}
			s := &Server{
				store: dbStore,
				// Do not replace this with a mock - these tests are used to test ProfileService as well
				profiles:      profiles.NewProfileService(evts, selectors.NewEnv()),
				providerStore: providers.NewProviderStore(dbStore),
				evt:           evts,
			}

			res, err := s.CreateProfile(ctx, tc.profile)
			if tc.wantErr != "" {
				niceErr, ok := err.(*util.NiceStatus)
				if !ok {
					t.Fatalf("Unexpected error type from CreateProfile: %v", err)
				}
				if niceErr.Details != tc.wantErr {
					t.Errorf("CreateProfile() error = %q, wantErr %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error from CreateProfile: %v", err)
			}

			profile, err := dbStore.GetProfileByNameAndLock(ctx, db.GetProfileByNameAndLockParams{
				Name:      tc.profile.GetProfile().GetName(),
				ProjectID: dbproj.ID,
			})
			if err != nil {
				t.Fatalf("Error getting profile: %v", err)
			}

			if tc.result.GetProfile().GetId() == "" {
				tc.result.GetProfile().Id = proto.String(profile.ID.String())
			}
			// For some reason, comparing these protos directly doesn't seem to work...
			if !proto.Equal(res, tc.result) {
				t.Errorf("CreateProfile() got = %v, want %v", res, tc.result)
			}
		})
	}
}

func ruleTypeName(prefix string, num int) string {
	return fmt.Sprintf("%s_rule_type_%d", prefix, num)
}

func ruleTypeSequenceCreate(
	ctx context.Context,
	dbStore db.Store,
	entity minderv1.Entity,
	projectID uuid.UUID,
	prefix string,
	count int,
) error {
	defstr := fmt.Sprintf("{\"in_entity\": \"%s\",\"ruleSchema\":{}}", entity.ToString())

	for i := 0; i < count; i++ {
		_, err := dbStore.CreateRuleType(ctx, db.CreateRuleTypeParams{
			Name:          ruleTypeName(prefix, i+1),
			ProjectID:     projectID,
			Definition:    []byte(defstr),
			SeverityValue: db.SeverityLow,
			ReleasePhase:  db.ReleaseStatusAlpha,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// nolint:gocyclo
func TestPatchProfile(t *testing.T) {
	t.Parallel()

	// We can't use mockdb.NewMockStore because BeginTransaction returns a *sql.Tx,
	// which is not an interface and can't be mocked.
	dbStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	if err != nil {
		t.Fatalf("Error creating fake store: %v", err)
	}

	// Common database setup
	ctx := context.Background()
	dbproj, err := dbStore.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Error creating project: %v", err)
	}

	err = ruleTypeSequenceCreate(ctx, dbStore, minderv1.Entity_ENTITY_REPOSITORIES, dbproj.ID, "repo", 4)
	if err != nil {
		t.Fatalf("Error creating rule type: %v", err)
	}

	err = ruleTypeSequenceCreate(ctx, dbStore, minderv1.Entity_ENTITY_PULL_REQUESTS, dbproj.ID, "pull", 4)
	if err != nil {
		t.Fatalf("Error creating rule type: %v", err)
	}

	err = ruleTypeSequenceCreate(ctx, dbStore, minderv1.Entity_ENTITY_ARTIFACTS, dbproj.ID, "artifact", 4)
	if err != nil {
		t.Fatalf("Error creating rule type: %v", err)
	}

	tests := []struct {
		name         string
		baseProfile  *minderv1.CreateProfileRequest
		patchRequest *minderv1.PatchProfileRequest
		wantErr      string
		result       *minderv1.PatchProfileResponse
	}{
		{
			name: "Patch profile with change to remediate",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:      "test_rem_patch",
					Remediate: proto.String("off"),
					Alert:     proto.String("off"),
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"remediate"}},
				Patch: &minderv1.Profile{
					Remediate: proto.String("on"),
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch",
					Remediate:   proto.String("on"),
					Alert:       proto.String("off"),
					DisplayName: "test_rem_patch",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
				},
			},
		},
		{
			name: "Patch profile with change to alert",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:      "test_rem_patch_alert",
					Remediate: proto.String("on"),
					Alert:     proto.String("on"),
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"alert"}},
				Patch: &minderv1.Profile{
					Alert: proto.String("off"),
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_alert",
					Remediate:   proto.String("on"),
					Alert:       proto.String("off"),
					DisplayName: "test_rem_patch_alert",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
				},
			},
		},
		{
			name: "Patch profile with repo rule addition",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_patch_repo_rule_add",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"repository"}},
				Patch: &minderv1.Profile{
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("repo", 2),
							Def:  &structpb.Struct{},
						},
					},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_add",
					Remediate:   proto.String("off"),
					Alert:       proto.String("on"),
					DisplayName: "test_rem_patch_repo_rule_add",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
						{
							Type: ruleTypeName("repo", 2),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 2),
						},
					},
				},
			},
		},
		{
			name: "Patch profile with repo rule deletion",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_patch_repo_rule_del",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("repo", 2),
							Def:  &structpb.Struct{},
						},
					},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"repository"}},
				Patch: &minderv1.Profile{
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_del",
					Remediate:   proto.String("off"),
					Alert:       proto.String("on"),
					DisplayName: "test_rem_patch_repo_rule_del",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
				},
			},
		},
		{
			name: "Patch profile with all allowed replacements",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_replace_all",
					Remediate:   proto.String("off"),
					Alert:       proto.String("off"),
					DisplayName: "display_test_rem_patch_repo_rule_replace_all",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("repo", 2),
							Def:  &structpb.Struct{},
						},
					},
					Artifact: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("artifact", 3),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("artifact", 4),
							Def:  &structpb.Struct{},
						},
					},
					PullRequest: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("pull", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("pull", 4),
							Def:  &structpb.Struct{},
						},
					},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{
					Paths: []string{
						"repository", "artifact", "pull_request",
						"remediate", "alert", "display_name",
					},
				},
				Patch: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_replace_all",
					Remediate:   proto.String("on"),
					Alert:       proto.String("on"),
					DisplayName: "dsp_test_rem_patch_repo_rule_replace_all",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 3),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("repo", 4),
							Def:  &structpb.Struct{},
						},
					},
					Artifact: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("artifact", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("artifact", 2),
							Def:  &structpb.Struct{},
						},
					},
					PullRequest: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("pull", 2),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("pull", 3),
							Def:  &structpb.Struct{},
						},
					},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_replace_all",
					Remediate:   proto.String("on"),
					Alert:       proto.String("on"),
					DisplayName: "dsp_test_rem_patch_repo_rule_replace_all",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 3),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 3),
						},
						{
							Type: ruleTypeName("repo", 4),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 4),
						},
					},
					Artifact: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("artifact", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("artifact", 1),
						},
						{
							Type: ruleTypeName("artifact", 2),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("artifact", 2),
						},
					},
					PullRequest: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("pull", 2),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("pull", 2),
						},
						{
							Type: ruleTypeName("pull", 3),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("pull", 3),
						},
					},
				},
			},
		},
		{
			name: "Patch profile with repo rule replacement",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_patch_repo_rule_replace",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
						},
						{
							Type: ruleTypeName("repo", 2),
							Def:  &structpb.Struct{},
						},
					},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"repository"}},
				Patch: &minderv1.Profile{
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 3),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 3),
						},
						{
							Type: ruleTypeName("repo", 4),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 4),
						},
					},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_rem_patch_repo_rule_replace",
					Remediate:   proto.String("off"),
					Alert:       proto.String("on"),
					DisplayName: "test_rem_patch_repo_rule_replace",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 3),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 3),
						},
						{
							Type: ruleTypeName("repo", 4),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 4),
						},
					},
				},
			},
		},
		{
			name: "Patch profile to add selectors",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_patch_add_selectors",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"selection"}},
				Patch: &minderv1.Profile{
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "repository",
							Selector:    "repository.name != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
					},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_patch_add_selectors",
					Remediate:   proto.String("off"),
					Alert:       proto.String("on"),
					DisplayName: "test_patch_add_selectors",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "repository",
							Selector:    "repository.name != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
					},
				},
			},
		},
		{
			name: "Patch profile to replace selectors",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_patch_replace_selectors",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
						},
					},
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "repository",
							Selector:    "repository.name != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
						{
							Entity:      "repository",
							Selector:    "repository.is_fork == false",
							Description: "No forks",
						},
					},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"selection"}},
				Patch: &minderv1.Profile{
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "repository",
							Selector:    "repository.name != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
						{
							Entity:      "repository",
							Selector:    "repository.is_private == false",
							Description: "No forks",
						},
					},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_patch_replace_selectors",
					Remediate:   proto.String("off"),
					Alert:       proto.String("on"),
					DisplayName: "test_patch_replace_selectors",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "repository",
							Selector:    "repository.name != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
						{
							Entity:      "repository",
							Selector:    "repository.is_private == false",
							Description: "No forks",
						},
					},
				},
			},
		},
		{
			name: "Patch profile to remove all selectors",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_patch_remove_selectors",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
						},
					},
					Selection: []*minderv1.Profile_Selector{
						{
							Entity:      "repository",
							Selector:    "repository.name != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
					},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"selection"}},
				Patch: &minderv1.Profile{
					Selection: []*minderv1.Profile_Selector{},
				},
			},
			result: &minderv1.PatchProfileResponse{
				Profile: &minderv1.Profile{
					Name:        "test_patch_remove_selectors",
					Remediate:   proto.String("off"),
					Alert:       proto.String("on"),
					DisplayName: "test_patch_remove_selectors",
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
							Name: ruleTypeName("repo", 1),
						},
					},
				},
			},
		},
		// Negative tests
		{
			name: "Profile patch does not allow changing name",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_no_name_change",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"name"}},
				Patch: &minderv1.Profile{
					Name: "test_rem_yes_name_change",
				},
			},
			result:  &minderv1.PatchProfileResponse{},
			wantErr: "cannot change profile name",
		},
		{
			name: "Profile patch does not allow changing version",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_no_version_change",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"version"}},
				Patch: &minderv1.Profile{
					Version: "2.0.0",
				},
			},
			result:  &minderv1.PatchProfileResponse{},
			wantErr: "profile version is invalid",
		},
		{
			name: "Profile patch does not allow changing type",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_no_type_change",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"type"}},
				Patch: &minderv1.Profile{
					Type: "not_a_profile",
				},
			},
			result:  &minderv1.PatchProfileResponse{},
			wantErr: "profile type is invalid",
		},
		{
			name: "Profile patch does not allow changing labels",
			baseProfile: &minderv1.CreateProfileRequest{
				Profile: &minderv1.Profile{
					Name: "test_rem_no_label_change",
					Repository: []*minderv1.Profile_Rule{{
						Type: ruleTypeName("repo", 1),
						Def:  &structpb.Struct{},
					}},
				},
			},
			patchRequest: &minderv1.PatchProfileRequest{
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				Patch: &minderv1.Profile{
					Labels: []string{"label1", "label2"},
				},
			},
			result:  &minderv1.PatchProfileResponse{},
			wantErr: "labels cannot be updated",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: dbproj.ID},
			})
			evts := &stubeventer.StubEventer{}
			s := &Server{
				store: dbStore,
				// Do not replace this with a mock - these tests are used to test ProfileService as well
				profiles:      profiles.NewProfileService(evts, selectors.NewEnv()),
				providerStore: providers.NewProviderStore(dbStore),
				evt:           evts,
			}

			baseProfile, err := s.CreateProfile(ctx, tc.baseProfile)
			require.NoError(t, err)

			tc.patchRequest.Id = baseProfile.GetProfile().GetId()
			tc.patchRequest.Context = baseProfile.GetProfile().GetContext()

			patchedProfile, err := s.PatchProfile(ctx, tc.patchRequest)
			if tc.wantErr != "" {
				niceErr, ok := err.(*util.NiceStatus)
				if !ok {
					t.Fatalf("Unexpected error type from PatchProfile: %v", err)
				}
				if !strings.Contains(niceErr.Details, tc.wantErr) {
					t.Errorf("PatchProfile() error = %q, wantErr %q", niceErr.Details, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error from PatchProfile: %v", err)
			}

			profile, err := dbStore.GetProfileByNameAndLock(ctx, db.GetProfileByNameAndLockParams{
				Name:      tc.baseProfile.GetProfile().GetName(),
				ProjectID: dbproj.ID,
			})
			if err != nil {
				t.Fatalf("Error getting profile: %v", err)
			}

			if err != nil {
				t.Fatalf("Error getting profile: %v", err)
			}
			if tc.result.GetProfile().GetId() == "" {
				tc.result.GetProfile().Id = proto.String(profile.ID.String())
			}

			tc.result.GetProfile().Context = tc.baseProfile.GetProfile().GetContext()

			// For some reason, comparing these protos directly doesn't seem to work...
			if !proto.Equal(patchedProfile, tc.result) {
				t.Errorf("PatchProfile() got = %+v\n want %+v", patchedProfile, tc.result)
			}
		})
	}
}

func TestPatchManagedProfile(t *testing.T) {
	t.Parallel()

	// We can't use mockdb.NewMockStore because BeginTransaction returns a *sql.Tx,
	// which is not an interface and can't be mocked.
	dbStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	if err != nil {
		t.Fatalf("Error creating fake store: %v", err)
	}

	// Common database setup
	ctx := context.Background()
	dbproj, err := dbStore.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Error creating project: %v", err)
	}

	err = dbStore.UpsertBundle(ctx, db.UpsertBundleParams{
		Name:      "test_managed_profile_bundle",
		Namespace: "testns",
	})
	require.NoError(t, err)

	dbBundle, err := dbStore.GetBundle(ctx, db.GetBundleParams{
		Name:      "test_managed_profile_bundle",
		Namespace: "testns",
	})
	require.NoError(t, err)

	dbSubscription, err := dbStore.CreateSubscription(ctx, db.CreateSubscriptionParams{
		ProjectID: dbproj.ID,
		BundleID:  dbBundle.ID,
	})
	require.NoError(t, err)

	dbProfile, err := dbStore.CreateProfile(ctx, db.CreateProfileParams{
		Name:           "test_managed_profile",
		ProjectID:      dbproj.ID,
		Alert:          db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
		SubscriptionID: uuid.NullUUID{UUID: dbSubscription.ID, Valid: true},
	})
	require.NoError(t, err)

	ctx = engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
		Project: engcontext.Project{ID: dbproj.ID},
	})

	evts := &stubeventer.StubEventer{}
	s := &Server{
		store: dbStore,
		// Do not replace this with a mock - these tests are used to test ProfileService as well
		profiles:      profiles.NewProfileService(evts, selectors.NewEnv()),
		providerStore: providers.NewProviderStore(dbStore),
		evt:           evts,
	}

	patchedProfile, err := s.PatchProfile(ctx, &minderv1.PatchProfileRequest{
		Id: dbProfile.ID.String(),
		Patch: &minderv1.Profile{
			Remediate: proto.String("on"),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "attempted to edit a rule type or profile which belongs to a bundle")
	require.Nil(t, patchedProfile)
}

func TestGetProfileStatusByName(t *testing.T) {
	t.Parallel()

	// Setup test database
	dbStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	require.NoError(t, err, "Error creating fake store")

	// Create a test project
	ctx := context.Background()
	dbproj, err := dbStore.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test-project",
		Metadata: []byte(`{}`),
	})
	require.NoError(t, err, "Error creating test project")

	// Create a test profile
	expectedProfileName := "test-profile"
	dbProfile, err := dbStore.CreateProfile(ctx, db.CreateProfileParams{
		Name:      expectedProfileName,
		ProjectID: dbproj.ID,
	})
	require.NoError(t, err, "Error creating test profile")

	// Setup context with project information
	ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
		Project: engcontext.Project{ID: dbproj.ID},
	})

	// Create server instance
	s := &Server{
		store: dbStore,
	}

	// Test case
	t.Run("Successful retrieval of profile status by name", func(t *testing.T) {
		t.Parallel()
		// Prepare request
		req := &minderv1.GetProfileStatusByNameRequest{
			Name: expectedProfileName,
		}

		// Call the method
		resp, err := s.GetProfileStatusByName(ctx, req)

		// Assertions
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, resp, "Response should not be nil")
		require.Equal(t, dbProfile.ID.String(), resp.ProfileStatus.ProfileId, "Profile ID should match")
		require.Equal(t, expectedProfileName, resp.ProfileStatus.ProfileName, "Profile name should match")
	})
}

func TestGetProfileStatusById(t *testing.T) {
	t.Parallel()

	// Setup test database
	dbStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	require.NoError(t, err, "Error creating fake store")

	// Create a test project
	ctx := context.Background()
	dbproj, err := dbStore.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test-project",
		Metadata: []byte(`{}`),
	})
	require.NoError(t, err, "Error creating test project")

	// Create a test profile
	expectedProfileName := "test-profile"
	dbProfile, err := dbStore.CreateProfile(ctx, db.CreateProfileParams{
		Name:      expectedProfileName,
		ProjectID: dbproj.ID,
	})
	require.NoError(t, err, "Error creating test profile")

	// Setup context with project information
	ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
		Project: engcontext.Project{ID: dbproj.ID},
	})

	// Create server instance
	s := &Server{
		store: dbStore,
	}

	// Test case
	t.Run("Successful retrieval of profile status by ID", func(t *testing.T) {
		t.Parallel()
		// Prepare request
		req := &minderv1.GetProfileStatusByIdRequest{
			Id: dbProfile.ID.String(),
		}

		// Call the method
		resp, err := s.GetProfileStatusById(ctx, req)

		// Assertions
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, resp, "Response should not be nil")
		require.Equal(t, dbProfile.ID.String(), resp.ProfileStatus.ProfileId, "Profile ID should match")
		require.Equal(t, expectedProfileName, resp.ProfileStatus.ProfileName, "Profile name should match")
	})
}

type deleteProfileTestCase struct {
	name    string
	req     *minderv1.DeleteProfileRequest
	id      uuid.UUID
	wantErr string
}

func setupDeleteProfileTest(t *testing.T) (db.Store, *db.Project, *db.Profile, *db.Profile, *db.Profile) {
	t.Helper()

	dbStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	require.NoError(t, err, "Error creating fake store")

	ctx := context.Background()
	dbproj, err := dbStore.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test",
		Metadata: []byte(`{}`),
	})
	require.NoError(t, err, "Error creating project")

	testProfile, err := dbStore.CreateProfile(ctx, db.CreateProfileParams{
		Name:      "test_profile",
		ProjectID: dbproj.ID,
		Alert:     db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
	})
	require.NoError(t, err, "Error creating test profile")

	namedProfile, err := dbStore.CreateProfile(ctx, db.CreateProfileParams{
		Name:      "named_profile",
		ProjectID: dbproj.ID,
		Alert:     db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
	})
	require.NoError(t, err, "Error creating test profile")

	err = dbStore.UpsertBundle(ctx, db.UpsertBundleParams{
		Name:      "test_bundle",
		Namespace: "testns",
	})
	require.NoError(t, err, "Error creating test bundle")

	dbBundle, err := dbStore.GetBundle(ctx, db.GetBundleParams{
		Name:      "test_bundle",
		Namespace: "testns",
	})
	require.NoError(t, err, "Error getting test bundle")

	dbSubscription, err := dbStore.CreateSubscription(ctx, db.CreateSubscriptionParams{
		ProjectID: dbproj.ID,
		BundleID:  dbBundle.ID,
	})
	require.NoError(t, err, "Error creating test subscription")

	bundleProfile, err := dbStore.CreateProfile(ctx, db.CreateProfileParams{
		Name:           "test_bundle_profile",
		ProjectID:      dbproj.ID,
		Alert:          db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
		SubscriptionID: uuid.NullUUID{UUID: dbSubscription.ID, Valid: true},
	})
	require.NoError(t, err, "Error creating test bundle profile")

	return dbStore, &dbproj, &testProfile, &namedProfile, &bundleProfile
}

func TestDeleteProfile(t *testing.T) {
	t.Parallel()

	dbStore, dbproj, testProfile, namedProfile, bundleProfile := setupDeleteProfileTest(t)
	otherUUID := uuid.New()

	tests := []deleteProfileTestCase{
		{
			name: "Delete existing profile",
			req: &minderv1.DeleteProfileRequest{
				Id: testProfile.ID.String(),
			},
		},
		{
			name: "Delete existing profile by name",
			req: &minderv1.DeleteProfileRequest{
				Id: namedProfile.Name,
			},
			id: namedProfile.ID,
		},
		{
			name: "Delete non-existent profile",
			req: &minderv1.DeleteProfileRequest{
				Id: otherUUID.String(),
			},
			wantErr: fmt.Sprintf("profile %q not found", otherUUID),
		},
		{
			name: "Delete with invalid profile ID",
			req: &minderv1.DeleteProfileRequest{
				Id: "not-a-uuid",
			},
			wantErr: `profile "not-a-uuid" not found`,
		},
		{
			name: "Delete bundle profile",
			req: &minderv1.DeleteProfileRequest{
				Id: bundleProfile.ID.String(),
			},
			wantErr: "cannot delete profile from bundle",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: dbproj.ID},
			})

			evts := &stubeventer.StubEventer{}
			s := &Server{
				store:         dbStore,
				profiles:      profiles.NewProfileService(evts, selectors.NewEnv()),
				providerStore: providers.NewProviderStore(dbStore),
				evt:           evts,
			}

			res, err := s.DeleteProfile(ctx, tc.req)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)

			id := tc.id
			if id == uuid.Nil {
				id = uuid.MustParse(tc.req.Id)
			}
			_, err = dbStore.GetProfileByID(ctx, db.GetProfileByIDParams{
				ID:        id,
				ProjectID: dbproj.ID,
			})
			require.ErrorIs(t, err, sql.ErrNoRows)
		})
	}
}

func TestListProfiles(t *testing.T) {
	t.Parallel()

	dbStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	if err != nil {
		t.Fatalf("Error creating fake store: %v", err)
	}

	ctx := context.Background()
	dbproj, err := dbStore.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Error creating project: %v", err)
	}

	testProfiles := []struct {
		name   string
		labels []string
	}{
		{name: "profile_c", labels: []string{"label1", "label2"}},
		{name: "profile_a", labels: []string{"label1"}},
		{name: "profile_b", labels: []string{"label2"}},
	}

	for _, tp := range testProfiles {
		_, err := dbStore.CreateProfile(ctx, db.CreateProfileParams{
			Name:      tp.name,
			ProjectID: dbproj.ID,
			Alert:     db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
			Labels:    tp.labels,
		})
		if err != nil {
			t.Fatalf("Error creating test profile %s: %v", tp.name, err)
		}
	}

	tests := []struct {
		name         string
		req          *minderv1.ListProfilesRequest
		wantErr      string
		wantProfiles []string
	}{
		{
			name: "List profiles with label filter 2",
			req: &minderv1.ListProfilesRequest{
				LabelFilter: "label2",
			},
			wantProfiles: []string{
				"profile_b",
				"profile_c",
			},
		},
		{
			name: "List profiles with label filter1",
			req: &minderv1.ListProfilesRequest{
				LabelFilter: "label1",
			},
			wantProfiles: []string{
				"profile_a",
				"profile_c",
			},
		},
		{
			name: "List profiles with non-existent label",
			req: &minderv1.ListProfilesRequest{
				LabelFilter: "non-existent",
			},
			wantProfiles: []string{},
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: dbproj.ID},
			})

			evts := &stubeventer.StubEventer{}
			s := &Server{
				store:         dbStore,
				profiles:      profiles.NewProfileService(evts, selectors.NewEnv()),
				providerStore: providers.NewProviderStore(dbStore),
				evt:           evts,
			}

			res, err := s.ListProfiles(ctx, tc.req)
			if tc.wantErr != "" {
				if err == nil {
					t.Errorf("ListProfiles() expected error containing %q, got nil", tc.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("ListProfiles() error = %v, wantErr %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("ListProfiles() unexpected error: %v", err)
				return
			}

			if res == nil {
				t.Error("Expected non-nil response")
				return
			}

			if len(res.Profiles) != len(tc.wantProfiles) {
				t.Errorf("ListProfiles() got %d profiles, want %d", len(res.Profiles), len(tc.wantProfiles))
				return
			}

			for i, wantName := range tc.wantProfiles {
				if res.Profiles[i].Name != wantName {
					t.Errorf("ListProfiles() profile[%d].Name = %q, want %q", i, res.Profiles[i].Name, wantName)
				}
			}
		})
	}
}

func TestGetProfileById(t *testing.T) {
	t.Parallel()

	dbStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	if err != nil {
		t.Fatalf("Error creating fake store: %v", err)
	}

	ctx := context.Background()
	dbproj, err := dbStore.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Error creating project: %v", err)
	}

	testProfile, err := dbStore.CreateProfile(ctx, db.CreateProfileParams{
		Name:      "test_profile",
		ProjectID: dbproj.ID,
		Alert:     db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
	})
	if err != nil {
		t.Fatalf("Error creating test profile: %v", err)
	}

	tests := []struct {
		name    string
		req     *minderv1.GetProfileByIdRequest
		wantErr string
	}{
		{
			name: "Get existing profile",
			req: &minderv1.GetProfileByIdRequest{
				Id: testProfile.ID.String(),
			},
		},
		{
			name: "Get non-existent profile",
			req: &minderv1.GetProfileByIdRequest{
				Id: uuid.New().String(),
			},
			wantErr: "profile not found",
		},
		{
			name: "Get with invalid profile ID",
			req: &minderv1.GetProfileByIdRequest{
				Id: "not-a-uuid",
			},
			wantErr: "invalid profile ID",
		},
		{
			name: "Get with empty profile ID",
			req: &minderv1.GetProfileByIdRequest{
				Id: "",
			},
			wantErr: "invalid profile ID",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: dbproj.ID},
			})

			evts := &stubeventer.StubEventer{}
			s := &Server{
				store:         dbStore,
				profiles:      profiles.NewProfileService(evts, selectors.NewEnv()),
				providerStore: providers.NewProviderStore(dbStore),
				evt:           evts,
			}

			res, err := s.GetProfileById(ctx, tc.req)
			if tc.wantErr != "" {
				if err == nil {
					t.Errorf("GetProfileById() expected error containing %q, got nil", tc.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("GetProfileById() error = %v, wantErr %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("GetProfileById() unexpected error: %v", err)
				return
			}

			if res == nil {
				t.Error("Expected non-nil response")
				return
			}

			if *(res.Profile.Id) != testProfile.ID.String() {
				t.Errorf("GetProfileById() profile.Id = %v, want %v", res.Profile.Id, testProfile.ID.String())
			}
			if res.Profile.Name != testProfile.Name {
				t.Errorf("GetProfileById() profile.Name = %v, want %v", res.Profile.Name, testProfile.Name)
			}
		})
	}
}

func TestGetProfileByName(t *testing.T) {
	t.Parallel()

	dbStore, cancelFunc, err := embedded.GetFakeStore()
	if cancelFunc != nil {
		t.Cleanup(cancelFunc)
	}
	if err != nil {
		t.Fatalf("Error creating fake store: %v", err)
	}

	ctx := context.Background()
	dbproj, err := dbStore.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test",
		Metadata: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("Error creating project: %v", err)
	}

	testProfile, err := dbStore.CreateProfile(ctx, db.CreateProfileParams{
		Name:      "test_profile",
		ProjectID: dbproj.ID,
		Alert:     db.NullActionType{ActionType: db.ActionTypeOn, Valid: true},
	})
	if err != nil {
		t.Fatalf("Error creating test profile: %v", err)
	}

	tests := []struct {
		name    string
		req     *minderv1.GetProfileByNameRequest
		wantErr string
	}{
		{
			name: "Get existing profile",
			req: &minderv1.GetProfileByNameRequest{
				Name: "test_profile",
			},
		},
		{
			name: "Get non-existent profile",
			req: &minderv1.GetProfileByNameRequest{
				Name: "non_existent_profile",
			},
			wantErr: "profile \"non_existent_profile\" not found",
		},
		{
			name: "Get with empty profile name",
			req: &minderv1.GetProfileByNameRequest{
				Name: "",
			},
			wantErr: "profile name must be specified",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project: engcontext.Project{ID: dbproj.ID},
			})

			evts := &stubeventer.StubEventer{}
			s := &Server{
				store:         dbStore,
				profiles:      profiles.NewProfileService(evts, selectors.NewEnv()),
				providerStore: providers.NewProviderStore(dbStore),
				evt:           evts,
			}

			res, err := s.GetProfileByName(ctx, tc.req)
			if tc.wantErr != "" {
				if err == nil {
					t.Errorf("GetProfileByName() expected error containing %q, got nil", tc.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("GetProfileByName() error = %v, wantErr %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("GetProfileByName() unexpected error: %v", err)
				return
			}

			if res == nil {
				t.Error("Expected non-nil response")
				return
			}

			if *(res.Profile.Id) != testProfile.ID.String() {
				t.Errorf("GetProfileByName() profile.Id = %v, want %v", res.Profile.Id, testProfile.ID.String())
			}
			if res.Profile.Name != testProfile.Name {
				t.Errorf("GetProfileByName() profile.Name = %v, want %v", res.Profile.Name, testProfile.Name)
			}
		})
	}
}
