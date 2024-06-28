// Copyright 2024 Stacklok, Inc.
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

package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/events"
	minderlogger "github.com/stacklok/minder/internal/logger"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	// DefaultExecutionTimeout is the timeout for execution of a set
	// of profiles on an entity.
	DefaultExecutionTimeout = 5 * time.Minute
	// ArtifactSignatureWaitPeriod is the waiting period for potential artifact signature to be available
	// before proceeding with evaluation.
	ArtifactSignatureWaitPeriod = 10 * time.Second
)

// ExecutorEventHandler is responsible for consuming entity events, passing
// entities to the executor, and then publishing the results.
type ExecutorEventHandler struct {
	evt                    events.Publisher
	handlerMiddleware      []message.HandlerMiddleware
	wgEntityEventExecution *sync.WaitGroup
	// terminationcontext is used to terminate the executor
	// when the server is shutting down.
	terminationcontext context.Context
	executor           Executor
}

// NewExecutorEventHandler creates the event handler for the executor
func NewExecutorEventHandler(
	ctx context.Context,
	evt events.Publisher,
	handlerMiddleware []message.HandlerMiddleware,
	executor Executor,
) *ExecutorEventHandler {
	return &ExecutorEventHandler{
		evt:                    evt,
		wgEntityEventExecution: &sync.WaitGroup{},
		terminationcontext:     ctx,
		handlerMiddleware:      handlerMiddleware,
		executor:               executor,
	}
}

// Register implements the Consumer interface.
func (e *ExecutorEventHandler) Register(r events.Registrar) {
	r.Register(events.TopicQueueEntityEvaluate, e.HandleEntityEvent, e.handlerMiddleware...)
}

// Wait waits for all the entity executions to finish.
func (e *ExecutorEventHandler) Wait() {
	e.wgEntityEventExecution.Wait()
}

// HandleEntityEvent handles events coming from webhooks/signals
// as well as the init event.
func (e *ExecutorEventHandler) HandleEntityEvent(msg *message.Message) error {
	// Grab the context before making a copy of the message
	msgCtx := msg.Context()
	// Let's not share memory with the caller
	msg = msg.Copy()

	inf, err := entities.ParseEntityEvent(msg)
	if err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	e.wgEntityEventExecution.Add(1)
	go func() {
		defer e.wgEntityEventExecution.Done()
		if inf.Type == pb.Entity_ENTITY_ARTIFACTS {
			time.Sleep(ArtifactSignatureWaitPeriod)
		}
		// TODO: Make this timeout configurable
		ctx, cancel := context.WithTimeout(e.terminationcontext, DefaultExecutionTimeout)
		defer cancel()

		ts := minderlogger.BusinessRecord(msgCtx)
		ctx = ts.WithTelemetry(ctx)

		logger := zerolog.Ctx(ctx)
		if err := inf.WithExecutionIDFromMessage(msg); err != nil {
			logger.Info().
				Str("message_id", msg.UUID).
				Msg("message does not contain execution ID, skipping")
			return
		}

		err := e.executor.EvalEntityEvent(ctx, inf)

		// record telemetry regardless of error. We explicitly record telemetry
		// here even though we also record it in the middleware because the evaluation
		// is done in a separate goroutine which usually still runs after the middleware
		// had already recorded the telemetry.
		logMsg := zerolog.Ctx(ctx).Info()
		if err != nil {
			logMsg = zerolog.Ctx(ctx).Error()
		}
		ts.Record(logMsg).Send()

		if err != nil {
			zerolog.Ctx(ctx).Info().
				Str("project", inf.ProjectID.String()).
				Str("provider_id", inf.ProviderID.String()).
				Str("entity", inf.Type.String()).
				Err(err).Msg("got error while evaluating entity event")
		}

		// We don't need to unset the execution ID because the event is going to be
		// deleted from the database anyway. The aggregator will take care of that.
		msg, err := inf.BuildMessage()
		if err != nil {
			logger.Err(err).Msg("error building message")
			return
		}

		// Publish the result of the entity evaluation
		if err := e.evt.Publish(events.TopicQueueEntityFlush, msg); err != nil {
			fmt.Printf("Hello5 %v", err)
			logger.Err(err).Msg("error publishing flush event")
		}
	}()

	return nil
}
