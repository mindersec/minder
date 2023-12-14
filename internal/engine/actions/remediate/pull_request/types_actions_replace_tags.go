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

	"github.com/go-git/go-git/v5/plumbing/filemode"
	fzconfig "github.com/stacklok/frizbee/pkg/config"
	"github.com/stacklok/frizbee/pkg/ghactions"
	"github.com/stacklok/frizbee/pkg/utils"
	"gopkg.in/yaml.v3"

	"github.com/stacklok/minder/internal/engine/interfaces"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

var _ fsModifier = (*frizbeeTagResolveModification)(nil)

type frizbeeTagResolveModification struct {
	fsChangeSet

	ghCli v1.GitHub
}

var _ modificationConstructor = newFrizbeeTagResolveModification

func newFrizbeeTagResolveModification(
	params *modificationConstructorParams,
) (fsModifier, error) { // nolint:unparam // we need to match the interface
	return &frizbeeTagResolveModification{
		fsChangeSet: fsChangeSet{
			fs: params.bfs,
		},
		ghCli: params.ghCli,
	}, nil
}

func (ftr *frizbeeTagResolveModification) createFsModEntries(ctx context.Context, _ interfaces.ActionsParams) error {
	entries := []*fsEntry{}

	err := ghactions.TraverseGitHubActionWorkflows(ftr.fs, ".github/workflows", func(path string, wflow *yaml.Node) error {
		m, err := ghactions.ModifyReferencesInYAML(ctx, ftr.ghCli, wflow, &fzconfig.GHActions{})
		if err != nil {
			return fmt.Errorf("failed to process YAML file %s: %w", path, err)
		}

		buf, err := utils.YAMLToBuffer(wflow)
		if err != nil {
			return fmt.Errorf("failed to convert YAML to buffer: %w", err)
		}

		if m {
			entries = append(entries, &fsEntry{
				Path:    path,
				Content: buf.String(),
				Mode:    filemode.Regular.String(),
			})
		}

		return nil
	})
	if err != nil {
		return err
	}

	ftr.entries = entries
	return nil
}

func (ftr *frizbeeTagResolveModification) modifyFs() ([]*fsEntry, error) {
	err := ftr.fsChangeSet.writeEntries()
	if err != nil {
		return nil, fmt.Errorf("cannot write entries: %w", err)
	}
	return ftr.entries, nil
}
