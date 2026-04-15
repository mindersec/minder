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

	// declarations
	projectID := uuid.New()
	providerID := uuid.New()
	repositoryID := uuid.New()
	executionID := uuid.New()

	parallelOps := 2

	// -- end expectations

	evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
		Driver: "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{
			BlockPublishUntilSubscriberAck: true,
		},
	})
	require.NoError(t, err, "failed to setup eventer")

	pq := testqueue.NewPassthroughQueue(t)
	queued := pq.GetQueue()

	go func() {
		t.Log("Running eventer")
		evt.Register(constants.TopicQueueEntityFlush, pq.Pass)
		err := evt.Run(context.Background())
		require.NoError(t, err, "failed to run eventer")
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
		}).WithID(repositoryID).
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
	)

	t.Log("waiting for eventer to start")
	<-evt.Running()

	msg, err := eiw.BuildMessage()
	require.NoError(t, err, "expected no error")

	// Run in the background, twice
	for i := 0; i < parallelOps; i++ {
		go func() {
			t.Log("Running entity event handler")
			require.NoError(t, handler.HandleEntityEvent(msg), "expected no error")
		}()
	}

	// expect flush
	for i := 0; i < parallelOps; i++ {
		t.Log("waiting for flush")
		result := <-queued
		require.NotNil(t, result)
		require.Equal(t, providerID.String(), msg.Metadata.Get(entities.ProviderIDEventKey))
		require.Equal(t, "repository", msg.Metadata.Get(entities.EntityTypeEventKey))
		require.Equal(t, projectID.String(), msg.Metadata.Get(entities.ProjectIDEventKey))
	}

	require.NoError(t, evt.Close(), "expected no error")

	t.Log("waiting for executor to finish")
	handler.Wait()
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

func TestExecutorEventHandler_CleanupOnPanic(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	projectID := uuid.New()
	providerID := uuid.New()
	repositoryID := uuid.New()
	executionID := uuid.New()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eiw := entities.NewEntityInfoWrapper().
		WithProviderID(providerID).
		WithProjectID(projectID).
		WithRepository(&minderv1.Repository{
			Name:     "test",
			RepoId:   123,
			CloneUrl: "github.com/foo/bar.git",
		}).WithID(repositoryID).
		WithExecutionID(executionID)

	executor := mockengine.NewMockExecutor(ctrl)
	// Make the executor panic to test panic recovery and cleanup
	executor.EXPECT().
		EvalEntityEvent(gomock.Any(), gomock.Eq(eiw)).
		Do(func(context.Context, *entities.EntityInfoWrapper) {
			panic("test panic in executor")
		})

	handler := engine.NewExecutorEventHandler(
		ctx,
		nil,
		[]message.HandlerMiddleware{},
		executor,
	)

	msg, err := eiw.BuildMessage()
	require.NoError(t, err, "expected no error building message")
	require.NoError(t, handler.HandleEntityEvent(msg), "expected no error from HandleEntityEvent")

	// Give time for the goroutine to execute, panic, recover, and cleanup
	time.Sleep(100 * time.Millisecond)
	handler.Wait()
}
