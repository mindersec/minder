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

// Package pull_request provides the pull request remediation engine
package pull_request

import (
	"bytes"
	"context" // #nosec G505 - we're not using sha1 for crypto, only to quickly compare contents
	"fmt"

	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// See minder.proto for more detailed documentation
const (
	// minderContentModification replaces the contents of a file with the given template
	minderContentModification = "minder.content"
	// minderFrizbeeTagResolve replaces a github action tag with the appropriate checksum
	minderFrizbeeTagResolve = "minder.actions.replace_tags_with_sha"
)

var _ fsModifier = (*contentModification)(nil)

type contentModification struct {
	fsChangeSet
	prCfg *pb.RuleType_Definition_Remediate_PullRequestRemediation
}

var _ modificationConstructor = newContentModification

func newContentModification(
	params *modificationConstructorParams,
) (fsModifier, error) {
	// validate
	if params.prCfg == nil {
		return nil, fmt.Errorf("pull request config cannot be nil")
	}

	if len(params.prCfg.Contents) == 0 {
		return nil, fmt.Errorf("pull request config contents cannot be empty")
	}

	for _, cnt := range params.prCfg.Contents {
		if cnt.Path == "" {
			return nil, fmt.Errorf("pull request config contents path cannot be empty")
		}
		if cnt.Content == "" {
			return nil, fmt.Errorf("pull request config contents content cannot be empty")
		}
	}

	entries, err := prConfigToEntries(params.prCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create PR entries: %w", err)
	}

	return &contentModification{
		prCfg: params.prCfg,
		fsChangeSet: fsChangeSet{
			entries: entries,
			fs:      params.bfs,
		},
	}, nil
}

func prConfigToEntries(prCfg *pb.RuleType_Definition_Remediate_PullRequestRemediation) ([]*fsEntry, error) {
	entries := make([]*fsEntry, len(prCfg.Contents))
	for i, cnt := range prCfg.Contents {
		contentTemplate, err := util.ParseNewTextTemplate(&cnt.Content, fmt.Sprintf("Content[%d]", i))
		if err != nil {
			return nil, fmt.Errorf("cannot parse content template (index %d): %w", i, err)
		}

		mode := ghModeNonExecFile
		if cnt.GetMode() != "" {
			mode = *cnt.Mode
		}

		entries[i] = &fsEntry{
			Path:            cnt.Path,
			Mode:            mode,
			contentTemplate: contentTemplate,
		}
	}

	return entries, nil
}

func (ca *contentModification) createFsModEntries(
	_ context.Context,
	params interfaces.ActionsParams,
) error {
	data := map[string]interface{}{
		"Params":  params.GetRule().Params.AsMap(),
		"Profile": params.GetRule().Def.AsMap(),
	}
	for i, entry := range ca.entries {
		content := new(bytes.Buffer)

		if err := entry.contentTemplate.Execute(content, data); err != nil {
			return fmt.Errorf("cannot execute content template (index %d): %w", i, err)
		}
		entry.Content = content.String()
	}

	return nil

}

func (ca *contentModification) modifyFs() ([]*fsEntry, error) {
	err := ca.fsChangeSet.writeEntries()
	if err != nil {
		return nil, fmt.Errorf("cannot write entries: %w", err)
	}

	return ca.entries, nil
}
