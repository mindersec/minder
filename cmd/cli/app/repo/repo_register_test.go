//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestMessageJsonEncoding(t *testing.T) {
	t.Parallel()
	jsonMessage := `{
		"repository":{"owner":"test","name":"a-test","repoId":4000000000},
		"context":{"provider":"github","project":"1234"}
	}`

	protoMessage := minderv1.RegisterRepositoryRequest{}
	err := protojson.Unmarshal([]byte(jsonMessage), &protoMessage)

	assert.NoError(t, err)
	assert.Equal(t, int64(4000000000), protoMessage.GetRepository().GetRepoId())
}
