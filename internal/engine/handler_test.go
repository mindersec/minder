// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/engine"
	"github.com/mindersec/minder/internal/engine/entities"
	mockengine "github.com/mindersec/minder/internal/engine/mock"
	"github.com/mindersec/minder/internal/util/testqueue"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

func TestExecutorEventHandler_handleEntityEvent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	projectID := uuid.New()
	providerID := uuid.New()
	repositoryID := uuid.New()
	executionID := uuid.New()

	parallelOps := 2

	evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
		Driver: "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{
			BlockPublishUntilSubscriberAck: true,
		},
	})
	require.NoError(t, err)

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	go func() {
		evt.Register(constants.TopicQueueEntityFlush, pq.Pass)
		err := evt.Run(context.Background())
		require.NoError(t, err)
	}()

	testTimeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	eiw := entities.NewEntityInfoWrapper().
		WithProviderID(providerID).
		WithProjectID(projectID).
		WithRepository(&minderv1.Repository{
			Name:     "test",
			RepoId:   123,
			CloneUrl: "github.com/foo/bar.git",
		}).
		WithID(repositoryID).
		WithExecutionID(executionID)

	executor := mockengine.NewMockExecutor(ctrl)
	for i := 0; i < parallelOps; i++ {
		executor.EXPECT().
			EvalEntityEvent(gomock.Any(), gomock.Eq(eiw)).
			Return(nil)
	}

	handler := engine.NewExecutorEventHandler(
		ctx,
		evt,
		[]message.HandlerMiddleware{},
		executor,
		engine.DefaultExecutionTimeout,
	)

	<-evt.Running()

	msg, err := eiw.BuildMessage()
	require.NoError(t, err)

	for i := 0; i < parallelOps; i++ {
		go func() {
			require.NoError(t, handler.HandleEntityEvent(msg))
		}()
	}

	for i := 0; i < parallelOps; i++ {
		result := <-queued
		require.NotNil(t, result)
		require.Equal(t, providerID.String(), msg.Metadata.Get(entities.ProviderIDEventKey))
		require.Equal(t, "repository", msg.Metadata.Get(entities.EntityTypeEventKey))
		require.Equal(t, projectID.String(), msg.Metadata.Get(entities.ProjectIDEventKey))
	}

	require.NoError(t, evt.Close())
	handler.Wait()
}

// Behavior-based timeout test
func TestExecutorEventHandler_RespectsTimeout(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	projectID := uuid.New()
	providerID := uuid.New()
	repositoryID := uuid.New()
	executionID := uuid.New()

	// Mock executor that simulates long-running work
	executor := mockengine.NewMockExecutor(ctrl)
	executor.EXPECT().
		EvalEntityEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ *entities.EntityInfoWrapper) error {
			select {
			case <-time.After(5 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})

	evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
		Driver: "go-channel",
	})
	require.NoError(t, err)

	handler := engine.NewExecutorEventHandler(
		context.Background(),
		evt,
		nil,
		executor,
		1*time.Second,
	)

	eiw := entities.NewEntityInfoWrapper().
		WithProviderID(providerID).
		WithProjectID(projectID).
		WithRepository(&minderv1.Repository{
			Name: "test",
		}).
		WithID(repositoryID).
		WithExecutionID(executionID)

	msg, err := eiw.BuildMessage()
	require.NoError(t, err)

	start := time.Now()

	require.NoError(t, handler.HandleEntityEvent(msg))
	handler.Wait()

	elapsed := time.Since(start)

	//Ensure execution timed out early
	require.Less(t, elapsed, 3*time.Second, "execution did not timeout as expected")
}

func TestExecutorEventHandler_ShutdownCancelsNewEvents(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	executor := mockengine.NewMockExecutor(ctrl)

	executor.EXPECT().
		EvalEntityEvent(gomock.Any(), gomock.Any()).
		Times(0)

	handler := engine.NewExecutorEventHandler(
		ctx,
		nil,
		[]message.HandlerMiddleware{},
		executor,
	)

	// Trigger shutdown
	cancel()

	time.Sleep(10 * time.Millisecond)
	msg := message.NewMessage("1", []byte("{}"))

	// Call handler AFTER shutdown
	err := handler.HandleEntityEvent(msg)
	require.NoError(t, err)

	// Give time in case something incorrectly executes
	time.Sleep(50 * time.Millisecond)

	handler.Wait()
}
