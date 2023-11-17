//
// Copyright 2023 Stacklok, Inc.
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

package eea_test

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/config"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/eea"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/events"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	providerName = "test-provider"
)

func TestAggregator(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var concurrentEvents int64 = 100

	projectID, repoID := createNeededEntities(ctx, t)

	evt, err := events.Setup(ctx, &config.EventConfig{
		Driver: "go-channel",
		GoChannel: config.GoChannelEventConfig{
			BufferSize:                     concurrentEvents,
			BlockPublishUntilSubscriberAck: true,
		},
	}, nil)
	require.NoError(t, err)

	// we'll wait 2 seconds for the lock to be available
	var eventThreshold int64 = 2

	aggr := eea.NewEEA(testQueries, evt, &config.AggregatorConfig{
		LockInterval: eventThreshold,
	})

	rateLimitedMessages := newTestPubSub()
	flushedMessages := newTestPubSub()

	rateLimitedMessageTopic := t.Name()

	// This tests that the middleware works as expected
	evt.Register(rateLimitedMessageTopic, rateLimitedMessages.Add, aggr.AggregateMiddleware)

	// This tests that flushing works as expected
	evt.Register(engine.FlushEntityEventTopic, aggr.FlushMessageHandler)

	// This tests that flushing sends messages to the executor engine
	evt.Register(engine.ExecuteEntityEventTopic, flushedMessages.Add, aggr.AggregateMiddleware)

	go func() {
		t.Log("Running eventer")
		err := evt.Run(ctx)
		assert.NoError(t, err, "expected no error when running eventer")
	}()
	defer evt.Close()

	inf := engine.NewEntityInfoWrapper().
		WithRepository(&minderv1.Repository{}).
		WithRepositoryID(repoID).
		WithProjectID(projectID).
		WithProvider(providerName)

	<-evt.Running()

	t.Log("Publishing events")
	var wg sync.WaitGroup
	for i := 0; i < int(concurrentEvents); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, err := inf.BuildMessage()
			require.NoError(t, err, "expected no error when building message")
			err = evt.Publish(rateLimitedMessageTopic, msg.Copy())
			require.NoError(t, err, "expected no error when publishing message")
		}()
	}

	wg.Wait()
	rateLimitedMessages.Wait()

	assert.Equal(t, int32(1), rateLimitedMessages.count.Load(), "expected only one message to be published")

	t.Log("Waiting for lock to be available")
	time.Sleep(time.Duration(eventThreshold) * time.Second)

	t.Log("Publishing flush events")
	var flushWg sync.WaitGroup
	for i := 0; i < int(concurrentEvents); i++ {
		flushWg.Add(1)
		go func() {
			defer flushWg.Done()
			msg, err := inf.BuildMessage()
			require.NoError(t, err, "expected no error when building message")

			err = evt.Publish(engine.FlushEntityEventTopic, msg.Copy())
			require.NoError(t, err, "expected no error when publishing message")
		}()
	}

	flushWg.Wait()
	flushedMessages.Wait()

	// flushing should only happen once
	assert.Equal(t, int32(1), flushedMessages.count.Load(), "expected only one message to be published")
}

func createNeededEntities(ctx context.Context, t *testing.T) (projID uuid.UUID, repoID uuid.UUID) {
	t.Helper()

	// setup project
	proj, err := testQueries.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test-project",
		Metadata: json.RawMessage("{}"),
	})
	require.NoError(t, err, "expected no error when creating project")

	// setup provider
	_, err = testQueries.CreateProvider(ctx, db.CreateProviderParams{
		Name:       providerName,
		ProjectID:  proj.ID,
		Implements: []db.ProviderType{db.ProviderTypeRest},
		Definition: json.RawMessage(`{}`),
	})
	require.NoError(t, err, "expected no error when creating provider")

	// setup repo
	repo, err := testQueries.CreateRepository(ctx, db.CreateRepositoryParams{
		ProjectID: proj.ID,
		Provider:  providerName,
		RepoName:  "test-repo",
		RepoOwner: "test-owner",
		RepoID:    123,
	})
	require.NoError(t, err, "expected no error when creating repo")

	return proj.ID, repo.ID
}

type testPubSub struct {
	// counts the number of messages added
	count            *atomic.Int32
	firstMessageOnce *sync.Once
	// allows us to wait for the first message to be added
	firstMessage chan struct{}
}

func newTestPubSub() *testPubSub {
	var count atomic.Int32
	return &testPubSub{
		count:            &count,
		firstMessage:     make(chan struct{}),
		firstMessageOnce: &sync.Once{},
	}
}

func (t *testPubSub) Wait() {
	<-t.firstMessage
}

func (t *testPubSub) Add(_ *message.Message) error {
	t.count.Add(1)
	t.firstMessageOnce.Do(func() {
		t.firstMessage <- struct{}{}
	})
	return nil
}
