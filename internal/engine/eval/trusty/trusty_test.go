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

	"github.com/stacklok/minder/internal/engine/eval/pr_actions"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/engine/models"
	mockgithub "github.com/stacklok/minder/internal/providers/github/mock"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
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
					Dependency: &models.Dependency{
						Ecosystem: models.PyPIDependency,
						Name:      "requests",
						Version:   "0.0.1",
					},
					trustyReply: &trustytypes.Reply{
						PackageName: "requests",
						PackageType: string(models.PyPIDependency),
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
					Dependency: &models.Dependency{
						Ecosystem: models.PyPIDependency,
						Name:      "requests",
						Version:   "0.0.1",
					},
					trustyReply: &trustytypes.Reply{
						PackageName: "requests",
						PackageType: string(models.PyPIDependency),
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
					Dependency: &models.Dependency{
						Ecosystem: models.PyPIDependency,
						Name:      "python-oauth",
						Version:   "0.0.1",
					},
					trustyReply: &trustytypes.Reply{
						PackageName: "requests",
						PackageType: string(models.PyPIDependency),
						Summary: trustytypes.ScoreSummary{
							Score: &sg,
						},
					},
				},
				{
					Dependency: &models.Dependency{
						Ecosystem: models.PyPIDependency,
						Name:      "requestts",
						Version:   "0.0.1",
					},
					trustyReply: &trustytypes.Reply{
						PackageName: "requests",
						PackageType: string(models.PyPIDependency),
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
				"hey": "you",
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
		sut     *engif.Result
		mustErr bool
	}{
		{name: "normal", sut: &engif.Result{Object: &models.PRDependencies{}}, mustErr: false},
		{name: "invalid-object", sut: &engif.Result{Object: context.Background()}, mustErr: true},
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
	ghProvider := mockgithub.NewMockGitHub(nil)
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
	dep := models.ContextualDependency{
		Dep: models.Dependency{
			Ecosystem: models.NPMDependency,
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
				Dependency: &models.Dependency{},
				Reasons:    []RuleViolationReason{},
				trustyReply: &trustytypes.Reply{
					Summary: trustytypes.ScoreSummary{},
				},
			},
		},
		{
			name: "normal-response",
			sut: dependencyAlternatives{
				Dependency: &models.Dependency{},
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
				Dependency: &models.Dependency{},
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
				Dependency: &models.Dependency{},
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
				Dependency: &models.Dependency{},
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
