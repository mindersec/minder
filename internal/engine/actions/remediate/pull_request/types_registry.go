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
	"io"

	"github.com/go-git/go-billy/v5"

	"github.com/stacklok/minder/internal/engine/interfaces"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
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
