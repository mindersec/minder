// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package pull_request

import (
	"context"
	"fmt"
	"io"

	"github.com/go-git/go-billy/v5"

	"github.com/mindersec/minder/internal/engine/interfaces"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/mindersec/minder/pkg/providers/v1"
)

type fsModifier interface {
	hash() (string, error)
	writeSummary(out io.Writer) error
	createFsModEntries(ctx context.Context, params interfaces.ActionsParams) error
	modifyFs() ([]*fsEntry, error)
}

type modificationConstructorParams struct {
	prCfg *pb.RuleType_Definition_Remediate_PullRequestRemediation
	ghCli v1.GitHub
	bfs   billy.Filesystem
	def   map[string]any
}

type modificationConstructor func(*modificationConstructorParams) (fsModifier, error)

type modificationRegistry map[string]modificationConstructor

func newModificationRegistry() modificationRegistry {
	return make(map[string]modificationConstructor)
}

func (mr modificationRegistry) register(name string, constructor modificationConstructor) {
	mr[name] = constructor
}

func (mr modificationRegistry) registerBuiltIn() {
	mr.register(minderContentModification, newContentModification)
	mr.register(minderFrizbeeTagResolve, newFrizbeeTagResolveModification)
	mr.register(minderYQEvaluate, newYqExecute)
}

func (mr modificationRegistry) getModification(
	name string,
	params *modificationConstructorParams,
) (fsModifier, error) {
	constructor, ok := mr[name]
	if !ok {
		return nil, fmt.Errorf("unknown modification type: %s", name)
	}

	return constructor(params)
}
