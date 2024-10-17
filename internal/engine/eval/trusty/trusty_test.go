// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package trusty

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/rs/zerolog"
	trustytypes "github.com/stacklok/trusty-sdk-go/pkg/types"
	"github.com/stretchr/testify/require"

	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/pr_actions"
	"github.com/mindersec/minder/internal/engine/eval/templates"
	pbinternal "github.com/mindersec/minder/internal/proto"
	mock_github "github.com/mindersec/minder/internal/providers/github/mock"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

func TestBuildEvalResult(t *testing.T) {
	t.Parallel()
	sg := float64(6.4)
	now := time.Now()
	for _, tc := range []struct {
		name        string
		sut         *summaryPrHandler
		expectedNil bool
		evalErr     bool
	}{
		{"normal", &summaryPrHandler{}, true, false},
		{"malicious-package", &summaryPrHandler{
			trackedAlternatives: []dependencyAlternatives{
				{
					Dependency: &pbinternal.Dependency{
						Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI,
						Name:      "requests",
						Version:   "0.0.1",
					},
					trustyReply: &trustytypes.Reply{
						PackageName: "requests",
						PackageType: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI.AsString(),
						Summary: trustytypes.ScoreSummary{
							Score: &sg,
						},
						PackageData: struct {
							Archived   bool                       `json:"archived"`
							Deprecated bool                       `json:"is_deprecated"`
							Malicious  *trustytypes.MaliciousData `json:"malicious"`
						}{
							Archived:   false,
							Deprecated: false,
							Malicious: &trustytypes.MaliciousData{
								Summary:   "malicuous",
								Published: &now,
							},
						},
					},
				},
			},
		}, false, true},
		{"low-scored-package", &summaryPrHandler{
			trackedAlternatives: []dependencyAlternatives{
				{
					Dependency: &pbinternal.Dependency{
						Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI,
						Name:      "requests",
						Version:   "0.0.1",
					},
					trustyReply: &trustytypes.Reply{
						PackageName: "requests",
						PackageType: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI.AsString(),
						Summary: trustytypes.ScoreSummary{
							Score: &sg,
						},
					},
				},
			},
		}, false, true},
		{"malicious-and-low-score", &summaryPrHandler{
			trackedAlternatives: []dependencyAlternatives{
				{
					Dependency: &pbinternal.Dependency{
						Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI,
						Name:      "python-oauth",
						Version:   "0.0.1",
					},
					trustyReply: &trustytypes.Reply{
						PackageName: "requests",
						PackageType: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI.AsString(),
						Summary: trustytypes.ScoreSummary{
							Score: &sg,
						},
					},
				},
				{
					Dependency: &pbinternal.Dependency{
						Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI,
						Name:      "requestts",
						Version:   "0.0.1",
					},
					trustyReply: &trustytypes.Reply{
						PackageName: "requests",
						PackageType: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI.AsString(),
						Summary: trustytypes.ScoreSummary{
							Score: &sg,
						},
						PackageData: struct {
							Archived   bool                       `json:"archived"`
							Deprecated bool                       `json:"is_deprecated"`
							Malicious  *trustytypes.MaliciousData `json:"malicious"`
						}{
							Archived:   false,
							Deprecated: false,
							Malicious: &trustytypes.MaliciousData{
								Summary:   "malicuous",
								Published: &now,
							},
						},
					},
				},
			},
		}, false, true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := buildEvalResult(tc.sut)
			if tc.expectedNil {
				require.Nil(t, res)
				return
			}
			require.Equal(t, tc.evalErr, fmt.Sprintf("%T", res) == "*errors.EvaluationError", fmt.Sprintf("%T", res))
		})
	}
}

func TestParseRuleConfig(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		sut     map[string]any
		mustErr bool
	}{
		{
			"normal", map[string]any{
				"action":           pr_actions.ActionSummary,
				"ecosystem_config": []ecosystemConfig{},
			}, false,
		},
		{
			"unsupported-action", map[string]any{
				"action":           pr_actions.Action("a"),
				"ecosystem_config": []ecosystemConfig{},
			}, true,
		},
		{
			"invalid-config", map[string]any{
				"ecosystem_config": []string{
					"hey",
				},
			}, true,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := parseRuleConfig(tc.sut)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
		})
	}
}

func TestReadPullRequestDependencies(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		sut     *interfaces.Result
		mustErr bool
	}{
		{name: "normal", sut: &interfaces.Result{Object: &pbinternal.PrDependencies{}}, mustErr: false},
		{name: "invalid-object", sut: &interfaces.Result{Object: context.Background()}, mustErr: true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps, err := readPullRequestDependencies(tc.sut)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, deps)
		})
	}
}

func TestNewTrustyEvaluator(t *testing.T) {
	ghProvider := mock_github.NewMockGitHub(nil)
	t.Parallel()
	for _, tc := range []struct {
		name    string
		prv     provifv1.GitHub
		mustErr bool
	}{
		{"normal", ghProvider, false},
		{"no-provider", nil, true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, err := NewTrustyEvaluator(context.Background(), tc.prv)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, c)
		})
	}
}

func TestClassifyDependency(t *testing.T) {
	t.Parallel()
	mkfloat := func(f float64) *float64 { return &f }

	ctx := context.Background()
	logger := zerolog.Ctx(ctx).With().Logger()
	dep := &pbinternal.PrDependencies_ContextualDependency{
		Dep: &pbinternal.Dependency{
			Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM,
			Name:      "test",
			Version:   "v0.0.1",
		},
	}
	for _, tc := range []struct {
		name       string
		score      *trustytypes.Reply
		config     *config
		mustFilter bool
		expected   *dependencyAlternatives
	}{
		{
			name: "normal-good-score",
			score: &trustytypes.Reply{
				PackageName: "test",
				PackageType: "npm",
				Summary: trustytypes.ScoreSummary{
					Score: mkfloat(6.4),
				},
			},
			config: defaultConfig(),
			expected: &dependencyAlternatives{
				Reasons: []RuleViolationReason{},
			},
			mustFilter: false,
		},
		{
			name: "normal-bad-score",
			score: &trustytypes.Reply{
				PackageName: "test",
				PackageType: "npm",
				Summary: trustytypes.ScoreSummary{
					Score: mkfloat(4.0),
				},
			},
			config: defaultConfig(),
			expected: &dependencyAlternatives{
				Reasons:     []RuleViolationReason{TRUSTY_LOW_SCORE},
				trustyReply: &trustytypes.Reply{},
			},
			mustFilter: true,
		},
		{
			name: "normal-malicious",
			score: &trustytypes.Reply{
				PackageName: "test",
				PackageType: "npm",
				Summary:     trustytypes.ScoreSummary{Score: mkfloat(8.0)},
				PackageData: trustytypes.PackageData{
					Archived:   false,
					Deprecated: false,
					Malicious: &trustytypes.MaliciousData{
						Summary: "it is malicious",
						Details: "some details",
					},
				},
			},
			config: defaultConfig(),
			expected: &dependencyAlternatives{
				Reasons:     []RuleViolationReason{TRUSTY_MALICIOUS_PKG},
				trustyReply: &trustytypes.Reply{},
			},
			mustFilter: true,
		},
		{
			name: "normal-lowactivity",
			score: &trustytypes.Reply{
				PackageName: "test",
				PackageType: "npm",
				Summary: trustytypes.ScoreSummary{
					Score: mkfloat(8.0),
					Description: map[string]any{
						"activity": float64(3.0),
					},
				},
			},
			config: defaultConfig(),
			expected: &dependencyAlternatives{
				Reasons:     []RuleViolationReason{TRUSTY_LOW_ACTIVITY},
				trustyReply: &trustytypes.Reply{},
			},
			mustFilter: true,
		},
		{
			name: "normal-low-provenance",
			score: &trustytypes.Reply{
				PackageName: "test",
				PackageType: "npm",
				Summary: trustytypes.ScoreSummary{
					Score: mkfloat(8.0),
					Description: map[string]any{
						"provenance": float64(3.0),
					},
				},
			},
			config: defaultConfig(),
			expected: &dependencyAlternatives{
				Reasons:     []RuleViolationReason{TRUSTY_LOW_PROVENANCE},
				trustyReply: &trustytypes.Reply{},
			},
			mustFilter: true,
		},
		{
			name: "nil-activity",
			score: &trustytypes.Reply{
				PackageName: "test",
				PackageType: "npm",
				Summary: trustytypes.ScoreSummary{
					Score: mkfloat(8.0),
					Description: map[string]any{
						"provenance": nil,
					},
				},
			},
			config: defaultConfig(),
			expected: &dependencyAlternatives{
				Reasons: []RuleViolationReason{},
			},
			mustFilter: false,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := summaryPrHandler{}
			classifyDependency(
				ctx, &logger, tc.score, tc.config, &handler, dep,
			)
			if !tc.mustFilter {
				require.Len(t, handler.trackedAlternatives, 0)
				return
			}
			require.Len(t, handler.trackedAlternatives, 1)
			require.Equal(
				t, tc.expected.Reasons,
				handler.trackedAlternatives[0].Reasons,
				handler.trackedAlternatives[0].Reasons,
			)
		})
	}
}

func TestBuildScoreMatrix(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		sut      dependencyAlternatives
		expected []templateScoreComponent
	}{
		{
			name: "no-description",
			sut: dependencyAlternatives{
				Dependency: &pbinternal.Dependency{},
				Reasons:    []RuleViolationReason{},
				trustyReply: &trustytypes.Reply{
					Summary: trustytypes.ScoreSummary{},
				},
			},
		},
		{
			name: "normal-response",
			sut: dependencyAlternatives{
				Dependency: &pbinternal.Dependency{},
				Reasons:    []RuleViolationReason{},
				trustyReply: &trustytypes.Reply{
					Summary: trustytypes.ScoreSummary{
						Description: map[string]any{
							"activity":      "a",
							"activity_user": "b",
							"provenance":    "c",
							"activity_repo": "d",
						},
					},
				},
			},
			expected: []templateScoreComponent{
				{Label: "Package activity", Value: "a"},
				{Label: "User activity", Value: "b"},
				{Label: "Provenance", Value: "c"},
				{Label: "Repository activity", Value: "d"},
			},
		},
		{
			name: "normal-response",
			sut: dependencyAlternatives{
				Dependency: &pbinternal.Dependency{},
				Reasons:    []RuleViolationReason{},
				trustyReply: &trustytypes.Reply{
					Summary: trustytypes.ScoreSummary{
						Description: map[string]any{
							"activity":      "a",
							"activity_user": "b",
							"provenance":    "c",
							"activity_repo": "d",
						},
					},
				},
			},
			expected: []templateScoreComponent{
				{Label: "Package activity", Value: "a"},
				{Label: "User activity", Value: "b"},
				{Label: "Provenance", Value: "c"},
				{Label: "Repository activity", Value: "d"},
			},
		},
		{
			name: "typosquatting-low",
			sut: dependencyAlternatives{
				Dependency: &pbinternal.Dependency{},
				Reasons:    []RuleViolationReason{},
				trustyReply: &trustytypes.Reply{
					Summary: trustytypes.ScoreSummary{
						Description: map[string]any{
							"typosquatting": float64(10),
						},
					},
				},
			},
			expected: []templateScoreComponent{},
		},
		{
			name: "typosquatting-high",
			sut: dependencyAlternatives{
				Dependency: &pbinternal.Dependency{},
				Reasons:    []RuleViolationReason{},
				trustyReply: &trustytypes.Reply{
					Summary: trustytypes.ScoreSummary{
						Description: map[string]any{
							"typosquatting": float64(1),
						},
					},
				},
			},
			expected: []templateScoreComponent{
				{Label: "Typosquatting", Value: "⚠️ Dependency may be trying to impersonate a well known package"},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scoreMatrix := buildScoreMatrix(tc.sut)
			require.Len(t, scoreMatrix, len(tc.expected))
			if len(tc.expected) == 0 {
				return
			}
			for i := range tc.expected {
				require.True(t, slices.Contains(scoreMatrix, tc.expected[i]))
			}
		})
	}
}

func TestReadPackageDescription(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		sut  *trustytypes.Reply
	}{
		{
			name: "normal",
			sut:  &trustytypes.Reply{},
		},
		{
			name: "no-provenance",
			sut: &trustytypes.Reply{
				Summary: trustytypes.ScoreSummary{
					Description: map[string]any{
						"provenance": 1,
					},
				},
			},
		},
		{
			name: "nil-response",
			sut:  nil,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data := readPackageDescription(tc.sut)
			require.NotNil(t, data)
			require.NotNil(t, data)
			_, ok := data["provenance"]
			require.True(t, ok)
			_, ok = data["activity"]
			require.True(t, ok)
		})
	}
}

func TestEvaluationDetailRendering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		msg     string
		msgArgs []any
		tmpl    string
		args    any
		error   string
		details string
	}{
		// trusty template
		{
			name: "trusty template with both malicious and low-scoring packages",
			msg:  "this is the message",
			tmpl: templates.TrustyTemplate,
			args: map[string]any{
				"maliciousPackages":  []string{"package1", "package2"},
				"lowScoringPackages": []string{"package2", "package3"},
			},
			error:   "evaluation failure: this is the message",
			details: "Malicious packages:\n* package1\n* package2\nPackages with a low Trusty score:\n* package2\n* package3",
		},
		{
			name: "trusty template with only malicious packages",
			msg:  "this is the message",
			tmpl: templates.TrustyTemplate,
			args: map[string]any{
				"maliciousPackages":  []string{"package1", "package2"},
				"lowScoringPackages": []string{},
			},
			error:   "evaluation failure: this is the message",
			details: "Malicious packages:\n* package1\n* package2",
		},
		{
			name: "trusty template with only low-scoring packages",
			msg:  "this is the message",
			tmpl: templates.TrustyTemplate,
			args: map[string]any{
				"maliciousPackages":  []string{},
				"lowScoringPackages": []string{"package2", "package3"},
			},
			error:   "evaluation failure: this is the message",
			details: "Packages with a low Trusty score:\n* package2\n* package3",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := evalerrors.NewDetailedErrEvaluationFailed(
				tt.tmpl,
				tt.args,
				tt.msg,
				tt.msgArgs...,
			)

			require.Equal(t, tt.error, err.Error())
			evalErr, ok := err.(*evalerrors.EvaluationError)
			require.True(t, ok)
			require.Equal(t, tt.details, evalErr.Details())
		})
	}
}

func defaultConfig() *config {
	return &config{
		Action:          defaultAction,
		EcosystemConfig: defaultEcosystemConfig,
	}
}
