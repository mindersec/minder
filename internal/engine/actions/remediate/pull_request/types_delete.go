// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request

import (
	"context"
	"fmt"

	"github.com/go-git/go-billy/v5/util"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/engine/interfaces"
	"github.com/mindersec/minder/internal/remediationparams"
)

const (
	// minderDeleteModification deletes files matching the given patterns
	minderDeleteModification = remediationparams.MethodDelete
)

var _ fsModifier = (*deleteModification)(nil)

type deleteModification struct {
	fsChangeSet
	config remediationparams.DeleteConfig
}

var _ modificationConstructor = newDeleteModification

func newDeleteModification(
	params *modificationConstructorParams,
) (fsModifier, error) {
	var confMap map[string]any
	if params.prCfg.GetParams() != nil {
		confMap = params.prCfg.Params.AsMap()
	}

	conf, err := remediationparams.ParseDeleteConfig(confMap)
	if err != nil {
		return nil, err
	}

	return &deleteModification{
		fsChangeSet: fsChangeSet{
			fs: params.bfs,
		},
		config: *conf,
	}, nil
}

func (d *deleteModification) createFsModEntries(
	ctx context.Context,
	_ proto.Message,
	_ interfaces.ActionsParams,
) error {
	for _, pattern := range d.config.Patterns {
		matches, err := util.Glob(d.fs, pattern.Pattern)
		if err != nil {
			return fmt.Errorf("cannot glob pattern %s: %w", pattern.Pattern, err)
		}

		for _, match := range matches {
			// Skip directories: billy's Remove fails on non-empty dirs
			info, err := d.fs.Stat(match)
			if err != nil {
				return fmt.Errorf("cannot stat %s: %w", match, err)
			}
			if info.IsDir() {
				zerolog.Ctx(ctx).
					Debug().
					Str("path", match).
					Msg("skipping directory match; minder.delete only removes files")
				continue
			}

			d.entries = append(d.entries, &fsEntry{
				Path:   match,
				Delete: true,
			})
		}
	}

	return nil
}

func (d *deleteModification) modifyFs() ([]*fsEntry, error) {
	if err := d.writeEntries(); err != nil {
		return nil, fmt.Errorf("cannot write entries: %w", err)
	}
	return d.entries, nil
}
