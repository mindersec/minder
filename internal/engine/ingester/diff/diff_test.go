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

// Package diff provides the diff rule data ingest engine
package diff

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	pbinternal "github.com/stacklok/minder/internal/proto"
	mock_github "github.com/stacklok/minder/internal/providers/github/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestGetEcosystemForFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		filename       string
		diffIngestCfg  *pb.DiffType
		expectedEcoSys DependencyEcosystem
	}{
		{
			name:     "Exact match",
			filename: "package-lock.json",
			diffIngestCfg: &pb.DiffType{
				Ecosystems: []*pb.DiffType_Ecosystem{
					{
						Name:    "npm",
						Depfile: "package-lock.json",
					},
				},
			},
			expectedEcoSys: DepEcosystemNPM,
		},
		{
			name:     "Wildcard match",
			filename: "/path/to/package-lock.json",
			diffIngestCfg: &pb.DiffType{
				Ecosystems: []*pb.DiffType_Ecosystem{
					{
						Name:    "npm",
						Depfile: fmt.Sprintf("%s%s", wildcard, "package-lock.json"),
					},
				},
			},
			expectedEcoSys: DepEcosystemNPM,
		},
		{
			name:     "Depfile without wildcard does not match subdirectory",
			filename: "/path/to/package-lock.json",
			diffIngestCfg: &pb.DiffType{
				Ecosystems: []*pb.DiffType_Ecosystem{
					{
						Name:    "npm",
						Depfile: "package-lock.json",
					},
				},
			},
			expectedEcoSys: DepEcosystemNPM,
		},
		{
			name:     "Wildcard not a match - wrong filename",
			filename: "/path/to/not-package-lock.json",
			diffIngestCfg: &pb.DiffType{
				Ecosystems: []*pb.DiffType_Ecosystem{
					{
						Name:    "npm",
						Depfile: fmt.Sprintf("%s/%s", wildcard, "package-lock.json"),
					},
				},
			},
			expectedEcoSys: DepEcosystemNone,
		},
		{
			name:     "No match",
			filename: "/path/to/README.md",
			diffIngestCfg: &pb.DiffType{
				Ecosystems: []*pb.DiffType_Ecosystem{
					{
						Name:    "npm",
						Depfile: fmt.Sprintf("%s%s", wildcard, "package-lock.json"),
					},
				},
			},
			expectedEcoSys: DepEcosystemNone,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			di := &Diff{
				cfg: tt.diffIngestCfg,
			}
			result := di.getEcosystemForFile(tt.filename)

			require.NotNil(t, result)
			assert.Equal(t, tt.expectedEcoSys, result)
		})
	}
}

func Test_setDifference(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		base    []int
		updated []int
		result  []int
	}{{
		name:    "empty updated",
		base:    []int{1, 2, 3},
		updated: []int{},
		result:  []int{},
	}, {
		name:    "empty base",
		base:    []int{},
		updated: []int{1, 2, 3},
		result:  []int{1, 2, 3},
	}, {
		name:    "no difference",
		base:    []int{1, 2, 3},
		updated: []int{3, 2, 1},
		result:  []int{},
	}, {
		name:    "add and remove",
		base:    []int{1, 2},
		updated: []int{2, 3},
		result:  []int{3},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := setDifference(tt.base, tt.updated, cmp.Compare)
			if !reflect.DeepEqual(got, tt.result) {
				t.Errorf("setDifference() = %v, want %v", got, tt.result)
			}
		})
	}
}

func Test_getScalibrTypeDiff(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		baseFiles        map[string]string
		updatedFiles     map[string]string
		expectedPackages []*pbinternal.PrDependencies_ContextualDependency
	}{{
		name: "no matching files",
		updatedFiles: map[string]string{
			"README.md": "words",
			"code.py":   "print('hello')",
			"main.go":   "package main",
		},
		expectedPackages: []*pbinternal.PrDependencies_ContextualDependency{},
	}, {
		name: "python diff",
		baseFiles: map[string]string{
			"requirements.txt": "requests==2.25.1\npandas==1.2.3\n",
		},
		updatedFiles: map[string]string{
			"requirements.txt": "pydantic>=1.7.1\nrequests==2.33.1\n",
		},
		expectedPackages: []*pbinternal.PrDependencies_ContextualDependency{
			{
				Dep: &pbinternal.Dependency{
					Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "pydantic",
					Version:   "1.7.1",
				},
				File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
					Name: "requirements.txt",
				},
			}, {
				Dep: &pbinternal.Dependency{
					Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_PYPI,
					Name:      "requests",
					Version:   "2.33.1",
				},
				File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
					Name: "requirements.txt",
				},
			},
		},
	}, {
		name: "npm diff",
		baseFiles: map[string]string{
			"package.json": `{"name":"foo","version":"0.1.0","dependencies":{
				"express":"4.17.1",
				"value-equal":"1.0.1"
			}}`,
			"package-lock.json": `{"name":"foo","version":"0.1.0","lockfileVersion":2,"packages":{
				"node_modules/express":{"version":"4.17.1"},
				"node_modules/value-equal":{"version":"1.0.1"}
			}}`,
		},
		updatedFiles: map[string]string{
			"package.json": `{"name":"foo","version":"0.1.0","dependencies":{
				"@babel/core":"7.12.1",
				"express":"4.17.3"
			}}`,
			"package-lock.json": `{"name":"foo","version":"0.1.0","lockfileVersion":2,"packages":{
				"node_modules/@babel/core":{"version":"7.12.1"},
				"node_modules/express":{"version":"4.17.3"}
			}}`,
		},
		expectedPackages: []*pbinternal.PrDependencies_ContextualDependency{
			{
				Dep: &pbinternal.Dependency{
					Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM,
					Name:      "@babel/core",
					Version:   "7.12.1",
				},
				File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
					Name: "package-lock.json",
				},
			}, {
				Dep: &pbinternal.Dependency{
					Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM,
					Name:      "express",
					Version:   "4.17.3",
				},
				File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
					Name: "package-lock.json",
				},
			},
		},
	}, {
		name: "go diff",
		baseFiles: map[string]string{
			"go.mod": "module test\ngo 1.23.1\nrequire (\n" +
				"github.com/gorilla/mux v1.8.0\n" +
				"github.com/x/mod v0.21.0 // indirect\n" +
				")",
		},
		updatedFiles: map[string]string{
			"go.mod": "module test\ngo 1.23.1\nrequire (\n" +
				"github.com/coreos/go-semver v0.3.1 // indirect\n" +
				"github.com/gorilla/mux v1.9.1\n" +
				")",
		},
		expectedPackages: []*pbinternal.PrDependencies_ContextualDependency{
			{
				Dep: &pbinternal.Dependency{
					Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_GO,
					Name:      "github.com/coreos/go-semver",
					Version:   "0.3.1",
				},
				File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
					Name: "go.mod",
				},
			}, {
				Dep: &pbinternal.Dependency{
					Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_GO,
					Name:      "github.com/gorilla/mux",
					Version:   "1.9.1",
				},
				File: &pbinternal.PrDependencies_ContextualDependency_FilePatch{
					Name: "go.mod",
				},
			},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() { ctrl.Finish() })
			ghClient := mock_github.NewMockGitHub(ctrl)

			differ, err := NewDiffIngester(&pb.DiffType{Type: pb.DiffTypeNewDeps}, ghClient)
			if err != nil {
				t.Fatalf("NewDiffIngester() error = %v", err)
			}

			req := &pb.PullRequest{
				Url:            "https://api.github.com/repos/evan-testing-minder/docs-test/pulls/2",
				CommitSha:      "5fab4eb53bdfdd879b841564ed9e8064de271cd2",
				Number:         2,
				RepoOwner:      "evan-testing-minder",
				RepoName:       "docs-test",
				AuthorId:       1234,
				Action:         "opened",
				BaseCloneUrl:   "https://github.com/evan-testing-minder/docs-test.git",
				TargetCloneUrl: "https://github.com/evankanderson/docs-test.git",
				BaseRef:        "main",
				TargetRef:      "fix-some-deps",
			}

			ghClient.EXPECT().Clone(gomock.Any(), req.BaseCloneUrl, req.BaseRef).Return(fakeClone(tt.baseFiles))
			ghClient.EXPECT().Clone(gomock.Any(), req.TargetCloneUrl, req.TargetRef).Return(fakeClone(tt.updatedFiles))

			result, err := differ.Ingest(context.Background(), req, nil)
			if err != nil {
				t.Fatalf("Ingest() error = %v", err)
			}

			packages, ok := result.Object.(*pbinternal.PrDependencies)
			if !ok {
				t.Fatalf("unexpected object type: %T", result.Object)
			}

			assert.Equal(t, tt.expectedPackages, packages.Deps)
		})
	}
}

// Returns a fake git.Repository which contains the specified files as a worktree
func fakeClone(files map[string]string) (*git.Repository, error) {
	fakeFs := memfs.New()
	repo, err := git.Init(memory.NewStorage(), fakeFs)
	if err != nil {
		return nil, err
	}
	// worktree, err := repo.Worktree()
	// if err != nil {
	// 	return nil, err
	// }

	for filename, content := range files {
		file, err := fakeFs.Create(filename)
		if err != nil {
			return nil, err
		}
		_, err = file.Write([]byte(content))
		if err != nil {
			return nil, err
		}
	}
	return repo, nil
}
