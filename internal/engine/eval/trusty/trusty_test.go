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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/engine/eval/pr_actions"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	mock_github "github.com/stacklok/minder/internal/providers/github/mock"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
					Dependency: &v1.Dependency{
						Ecosystem: v1.DepEcosystem_DEP_ECOSYSTEM_PYPI,
						Name:      "requests",
						Version:   "0.0.1",
					},
					trustyReply: &Reply{
						PackageName: "requests",
						PackageType: v1.DepEcosystem_DEP_ECOSYSTEM_PYPI.AsString(),
						Summary: ScoreSummary{
							Score: &sg,
						},
						PackageData: struct {
							Archived   bool           `json:"archived"`
							Deprecated bool           `json:"is_deprecated"`
							Malicious  *MaliciousData `json:"malicious"`
						}{
							Archived:   false,
							Deprecated: false,
							Malicious: &MaliciousData{
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
					Dependency: &v1.Dependency{
						Ecosystem: v1.DepEcosystem_DEP_ECOSYSTEM_PYPI,
						Name:      "requests",
						Version:   "0.0.1",
					},
					trustyReply: &Reply{
						PackageName: "requests",
						PackageType: v1.DepEcosystem_DEP_ECOSYSTEM_PYPI.AsString(),
						Summary: ScoreSummary{
							Score: &sg,
						},
					},
				},
			},
		}, false, true},
		{"malicious-and-low-score", &summaryPrHandler{
			trackedAlternatives: []dependencyAlternatives{
				{
					Dependency: &v1.Dependency{
						Ecosystem: v1.DepEcosystem_DEP_ECOSYSTEM_PYPI,
						Name:      "python-oauth",
						Version:   "0.0.1",
					},
					trustyReply: &Reply{
						PackageName: "requests",
						PackageType: v1.DepEcosystem_DEP_ECOSYSTEM_PYPI.AsString(),
						Summary: ScoreSummary{
							Score: &sg,
						},
					},
				},
				{
					Dependency: &v1.Dependency{
						Ecosystem: v1.DepEcosystem_DEP_ECOSYSTEM_PYPI,
						Name:      "requestts",
						Version:   "0.0.1",
					},
					trustyReply: &Reply{
						PackageName: "requests",
						PackageType: v1.DepEcosystem_DEP_ECOSYSTEM_PYPI.AsString(),
						Summary: ScoreSummary{
							Score: &sg,
						},
						PackageData: struct {
							Archived   bool           `json:"archived"`
							Deprecated bool           `json:"is_deprecated"`
							Malicious  *MaliciousData `json:"malicious"`
						}{
							Archived:   false,
							Deprecated: false,
							Malicious: &MaliciousData{
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
		{name: "normal", sut: &engif.Result{Object: &v1.PrDependencies{}}, mustErr: false},
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
