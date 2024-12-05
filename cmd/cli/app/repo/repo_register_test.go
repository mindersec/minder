// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
