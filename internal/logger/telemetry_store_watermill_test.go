// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/util/testqueue"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestTelemetryStoreWMMiddlewareLogsRepositoryInfo(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	l := zerolog.New(&buf)

	mdw := logger.NewTelemetryStoreWMMiddleware(&l)

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	evt, err := events.Setup(context.Background(), &serverconfig.EventConfig{
		Driver: "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{
			BlockPublishUntilSubscriberAck: true,
		},
	})
	require.NoError(t, err)

	go func() {
		t.Log("Running eventer")
		evt.Register("test-topic", pq.Pass, events.WebhookSubscriber, mdw.TelemetryStoreMiddleware)
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	providerName := "test-provider"
	projectID := uuid.New()
	repositoryID := uuid.New()

	eiw := entities.NewEntityInfoWrapper().
		WithProvider(providerName).
		WithProjectID(projectID).
		WithRepository(&minderv1.Repository{
			Name:     "test",
			RepoId:   123,
			CloneUrl: "github.com/foo/bar.git",
		}).WithRepositoryID(repositoryID)

	msg, err := eiw.BuildMessage()
	require.NoError(t, err, "expected no error")

	require.NoError(t, evt.Publish("test-topic", msg), "expected no error")
	require.NotNil(t, <-queued, "expected message")
	require.NoError(t, evt.Close(), "expected no error")

	logged := map[string]any{}

	require.NoError(t, json.Unmarshal(buf.Bytes(), &logged), "expected no error")

	t.Logf("logged: %v", logged)

	require.Equal(t, projectID.String(), logged["project"], "expected project ID to be logged")
	require.Equal(t, providerName, logged["provider"], "expected provider to be logged")
	require.Equal(t, repositoryID.String(), logged["repository"], "expected repository ID to be logged")
	require.Equal(t, "true", logged["telemetry"], "expected telemetry to be logged")
}
