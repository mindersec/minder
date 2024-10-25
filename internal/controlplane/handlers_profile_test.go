// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
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
			name: "Selector with valid but unsupported entity",
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
							// this entity is valid in the sense that it converts to an entity type but
							// the current selelectors implementation does not support it
							Entity:      "build_environment",
							Selector:    "repository.name != 'stacklok/demo-repo-go'",
							Description: "Exclude stacklok/demo-repo-go",
						},
					},
				},
			},
			wantErr: `unsupported entity type: no environment for entity ENTITY_BUILD_ENVIRONMENTS: unsupported entity type`,
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
					Repository: []*minderv1.Profile_Rule{
						{
							Type: ruleTypeName("repo", 1),
							Def:  &structpb.Struct{},
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

			// Create the base profile
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
