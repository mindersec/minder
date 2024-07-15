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
	"slices"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/engcontext"
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
	executor               Executor
	// cancels are a set of cancel functions for current entity events in flight.
	// This allows us to cancel rule evaluation directly when terminationContext
	// is cancelled.
	cancels []*context.CancelFunc
	lock    sync.Mutex
}

// NewExecutorEventHandler creates the event handler for the executor
func NewExecutorEventHandler(
	ctx context.Context,
	evt events.Publisher,
	handlerMiddleware []message.HandlerMiddleware,
	executor Executor,
) *ExecutorEventHandler {
	eh := &ExecutorEventHandler{
		evt:                    evt,
		wgEntityEventExecution: &sync.WaitGroup{},
		handlerMiddleware:      handlerMiddleware,
		executor:               executor,
	}
	go func() {
		<-ctx.Done()
		eh.lock.Lock()
		defer eh.lock.Unlock()

		for _, cancel := range eh.cancels {
			(*cancel)()
		}
	}()

	return eh
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

	// NOTE: we're _deliberately_ "escaping" from the parent context's Cancel/Done
	// completion, because the default watermill behavior for both Go channels and
	// SQL is to process messages sequentially, but we need additional parallelism
	// beyond that.  When we switch to a different message processing system, we
	// should aim to remove this goroutine altogether and have the messaging system
	// provide the parallelism.
	// We _do_ still want to cancel on shutdown, however.
	// TODO: Make this timeout configurable
	msgCtx := context.WithoutCancel(msg.Context())
	msgCtx, shutdownCancel := context.WithCancel(msgCtx)

	e.lock.Lock()
	e.cancels = append(e.cancels, &shutdownCancel)
	e.lock.Unlock()

	// Let's not share memory with the caller.  Note that this does not copy Context
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

		ctx, cancel := context.WithTimeout(msgCtx, DefaultExecutionTimeout)
		defer cancel()
		defer func() {
			e.lock.Lock()
			e.cancels = slices.DeleteFunc(e.cancels, func(cf *context.CancelFunc) bool {
				return cf == &shutdownCancel
			})
			e.lock.Unlock()
		}()

		ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
			Project: engcontext.Project{ID: inf.ProjectID},
			// TODO: extract Provider name from ProviderID?
		})

		ts := minderlogger.BusinessRecord(ctx)
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
		logMsg := logger.Info()
		if err != nil {
			logMsg = logger.Error()
		}
		ts.Record(logMsg).Send()

		if err != nil {
			logger.Info().
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
			logger.Err(err).Msg("error publishing flush event")
		}
	}()

	return nil
}
