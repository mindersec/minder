// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/util/testqueue"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer"
)

func TestTelemetryStoreWMMiddlewareLogsRepositoryInfo(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	l := zerolog.New(&buf)

	mdw := logger.NewTelemetryStoreWMMiddleware(&l)

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	evt, err := eventer.New(context.Background(), &serverconfig.EventConfig{
		Driver: "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{
			BlockPublishUntilSubscriberAck: true,
		},
	})
	require.NoError(t, err)

	go func() {
		t.Log("Running eventer")
		evt.Register("test-topic", pq.Pass, mdw.TelemetryStoreMiddleware)
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
	}()

	<-evt.Running()

	providerID := uuid.New()
	projectID := uuid.New()
	repositoryID := uuid.New()

	eiw := entities.NewEntityInfoWrapper().
		WithProviderID(providerID).
		WithProjectID(projectID).
		WithRepository(&minderv1.Repository{
			Name:     "test",
			RepoId:   123,
			CloneUrl: "github.com/foo/bar.git",
		}).WithID(repositoryID)

	msg, err := eiw.BuildMessage()
	require.NoError(t, err, "expected no error")

	require.NoError(t, evt.Publish("test-topic", msg), "expected no error")
	require.NotNil(t, <-queued, "expected message")
	require.NoError(t, evt.Close(), "expected no error")

	logged := map[string]any{}

	require.NoError(t, json.Unmarshal(buf.Bytes(), &logged), "expected no error")

	t.Logf("logged: %v", logged)

	require.Equal(t, projectID.String(), logged["project"], "expected project ID to be logged")
	require.Equal(t, providerID.String(), logged["provider_id"], "expected provider to be logged")
	require.Equal(t, repositoryID.String(), logged["repository"], "expected repository ID to be logged")
	require.Equal(t, "true", logged["telemetry"], "expected telemetry to be logged")
}
