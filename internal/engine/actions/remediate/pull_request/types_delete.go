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
)

const (
	// minderDeleteModification deletes files matching the given patterns
	minderDeleteModification = "minder.delete"
)

var _ fsModifier = (*deleteModification)(nil)

type deleteConfig struct {
	Patterns []struct {
		Pattern string `json:"pattern"`
		Type    string `json:"type"`
	} `json:"patterns"`
}

type deleteModification struct {
	fsChangeSet
	config deleteConfig
}

var _ modificationConstructor = newDeleteModification

func newDeleteModification(
	params *modificationConstructorParams,
) (fsModifier, error) {
	if params.prCfg.GetParams() == nil {
		return nil, fmt.Errorf("delete modification requires params")
	}

	confMap := params.prCfg.Params.AsMap()
	patternsRaw, ok := confMap["patterns"]
	if !ok {
		return nil, fmt.Errorf("delete modification requires patterns")
	}

	patternsList, ok := patternsRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("patterns must be a list")
	}

	var conf deleteConfig
	for i, p := range patternsList {
		pm, ok := p.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("patterns[%d] must be an object", i)
		}
		pattern, _ := pm["pattern"].(string)
		typ, _ := pm["type"].(string)
		if pattern == "" {
			return nil, fmt.Errorf("patterns[%d] missing required field: pattern", i)
		}
		if typ == "" {
			return nil, fmt.Errorf("patterns[%d] missing required field: type", i)
		}
		if typ != string(patternTypeGlob) {
			return nil, fmt.Errorf("patterns[%d] unsupported type %q (only %q is supported)", i, typ, patternTypeGlob)
		}
		conf.Patterns = append(conf.Patterns, struct {
			Pattern string `json:"pattern"`
			Type    string `json:"type"`
		}{Pattern: pattern, Type: typ})
	}

	if len(conf.Patterns) == 0 {
		return nil, fmt.Errorf("delete modification requires at least one valid pattern")
	}

	return &deleteModification{
		fsChangeSet: fsChangeSet{
			fs: params.bfs,
		},
		config: conf,
	}, nil
}

func (d *deleteModification) createFsModEntries(
	ctx context.Context,
	_ proto.Message,
	_ interfaces.ActionsParams,
) error {
	for _, pattern := range d.config.Patterns {
		if pattern.Type != string(patternTypeGlob) {
			zerolog.Ctx(ctx).
				Warn().
				Str("pattern.Type", pattern.Type).
				Msg("unsupported pattern type")
			continue
		}

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
