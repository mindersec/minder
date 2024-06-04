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

package pull_request

import (
	"context"
	"fmt"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/stacklok/frizbee/pkg/replacer"
	"github.com/stacklok/frizbee/pkg/utils/config"

	"github.com/stacklok/minder/internal/engine/interfaces"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

var _ fsModifier = (*frizbeeTagResolveModification)(nil)

type frizbeeTagResolveModification struct {
	fsChangeSet

	fzcfg *config.GHActions

	ghCli v1.GitHub
}

var _ modificationConstructor = newFrizbeeTagResolveModification

func newFrizbeeTagResolveModification(
	params *modificationConstructorParams,
) (fsModifier, error) {
	exclude := []string{}
	if ex := parseExcludesFromRepoConfig(params.bfs); ex != nil {
		exclude = ex
	} else if ex := parseExcludeFromDef(params.def); ex != nil {
		exclude = ex
	} else if ex := params.prCfg.GetActionsReplaceTagsWithSha().GetExclude(); ex != nil {
		exclude = ex
	}
	return &frizbeeTagResolveModification{
		fsChangeSet: fsChangeSet{
			fs: params.bfs,
		},
		fzcfg: &config.GHActions{
			Filter: config.Filter{
				Exclude: exclude,
			},
		},
		ghCli: params.ghCli,
	}, nil
}

func (ftr *frizbeeTagResolveModification) createFsModEntries(ctx context.Context, _ interfaces.ActionsParams) error {
	// Create a new Frizbee instance
	r := replacer.NewGitHubActionsReplacer(&config.Config{GHActions: *ftr.fzcfg}).WithGitHubClient(ftr.ghCli)

	// Parse the .github/workflows directory and replace tags with digests
	ret, err := r.ParsePathInFS(ctx, ftr.fs, ".github/workflows")
	if err != nil {
		return fmt.Errorf("failed to parse path in filesystem: %w", err)
	}

	// Add the modified paths and contents to the fsChangeSet, if any
	for modifiedPath, modifiedContent := range ret.Modified {
		ftr.entries = append(ftr.entries, &fsEntry{
			Path:    modifiedPath,
			Content: modifiedContent,
			Mode:    filemode.Regular.String(),
		})
	}
	// All good
	return nil
}

func (ftr *frizbeeTagResolveModification) modifyFs() ([]*fsEntry, error) {
	err := ftr.fsChangeSet.writeEntries()
	if err != nil {
		return nil, fmt.Errorf("cannot write entries: %w", err)
	}
	return ftr.entries, nil
}

func parseExcludeFromDef(def map[string]any) []string {
	if def == nil {
		return nil
	}

	exclude, ok := def["exclude"]
	if !ok {
		return nil
	}

	excludeSlice, ok := exclude.([]interface{})
	if !ok {
		return nil
	}

	excludeStrings := []string{}
	for _, ex := range excludeSlice {
		excludeStr, ok := ex.(string)
		if !ok {
			return nil
		}

		excludeStrings = append(excludeStrings, excludeStr)
	}

	return excludeStrings
}

func parseExcludesFromRepoConfig(fs billy.Filesystem) []string {
	for _, fname := range []string{".frizbee.yml", ".frizbee.yaml"} {
		cfg, err := config.ParseConfigFileFromFS(fs, fname)
		if err != nil {
			continue
		}

		return cfg.GHActions.Filter.Exclude
	}
	return nil
}
