// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/engine/entities"
	minderlogger "github.com/mindersec/minder/internal/logger"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/eventer/constants"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

const (
	// DefaultExecutionTimeout is the timeout for execution of a set
	// of profiles on an entity.
	DefaultExecutionTimeout = 5 * time.Minute

	// ArtifactSignatureWaitPeriod is the waiting period for potential artifact signature to be available
	// before proceeding with evaluation.
	ArtifactSignatureWaitPeriod = 10 * time.Second
)

// ExecutorEventHandler handles entity events, executes evaluations,
// and publishes the results.
type ExecutorEventHandler struct {
	evt                    interfaces.Publisher
	handlerMiddleware      []message.HandlerMiddleware
	wgEntityEventExecution *sync.WaitGroup
	executor               Executor

	executionTimeout time.Duration

	cancels []*context.CancelFunc
	lock    sync.Mutex
}

// NewExecutorEventHandler creates a new ExecutorEventHandler with
// configurable execution timeout.
func NewExecutorEventHandler(
	ctx context.Context,
	evt interfaces.Publisher,
	handlerMiddleware []message.HandlerMiddleware,
	executor Executor,
	executionTimeout time.Duration,
) *ExecutorEventHandler {

	if executionTimeout <= 0 {
		executionTimeout = DefaultExecutionTimeout
	}

	eh := &ExecutorEventHandler{
		evt:                    evt,
		wgEntityEventExecution: &sync.WaitGroup{},
		handlerMiddleware:      handlerMiddleware,
		executor:               executor,
		executionTimeout:       executionTimeout,
	}

	zerolog.Ctx(ctx).Debug().
		Dur("execution_timeout", executionTimeout).
		Msg("executor event handler initialized")

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

// Register registers the handler for entity evaluation events.
func (e *ExecutorEventHandler) Register(r interfaces.Registrar) {
	r.Register(constants.TopicQueueEntityEvaluate, e.HandleEntityEvent, e.handlerMiddleware...)
}

// Wait blocks until all entity event executions are complete.
func (e *ExecutorEventHandler) Wait() {
	e.wgEntityEventExecution.Wait()
}

// HandleEntityEvent processes incoming entity events and triggers evaluation.
func (e *ExecutorEventHandler) HandleEntityEvent(msg *message.Message) error {

	// NOTE: we're _deliberately_ "escaping" from the parent context's Cancel/Done
	// completion, because the default watermill behavior for both Go channels and
	// SQL is to process messages sequentially, but we need additional parallelism
	// beyond that. When we switch to a different message processing system, we
	// should aim to remove this goroutine altogether and have the messaging system
	// provide the parallelism.
	// We _do_ still want to cancel on shutdown, however.
	msgCtx := context.WithoutCancel(msg.Context())

	// This allows us to cancel rule evaluation directly when terminationContext
	// is cancelled.
	//nolint:gosec
	msgCtx, shutdownCancel := context.WithCancel(msgCtx)

	e.lock.Lock()
	e.cancels = append(e.cancels, &shutdownCancel)
	e.lock.Unlock()

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

		// use configurable timeout
		ctx, cancel := context.WithTimeout(msgCtx, e.executionTimeout)
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
			Provider: engcontext.Provider{
				Name: inf.ProviderID.String(), // unchanged
			},
		})

		ts := minderlogger.BusinessRecord(ctx)
		ctx = ts.WithTelemetry(ctx)

		logger := zerolog.Ctx(ctx)

		if err := inf.WithExecutionIDFromMessage(msg); err != nil {
			logger.Debug().
				Str("message_id", msg.UUID).
				Msg("message does not contain execution ID, skipping")
			return
		}

		err := e.executor.EvalEntityEvent(ctx, inf)

		logMsg := logger.Info()
		if err != nil {
			logMsg = logger.Error()
		}
		ts.Record(logMsg).Send()

		// record telemetry regardless of error. We explicitly record telemetry
		// here even though we also record it in the middleware because the evaluation
		// is done in a separate goroutine which usually still runs after the middleware
		// had already recorded the telemetry.

		if err != nil {
			logger.Info().
				Str("project", inf.ProjectID.String()).
				Str("provider_id", inf.ProviderID.String()).
				Str("entity", inf.Type.String()).
				Str("entity_id", inf.EntityID.String()).
				Err(err).
				Msg("got error while evaluating entity event")
		}

		msg, err := inf.BuildMessage()
		if err != nil {
			logger.Err(err).Msg("error building message")
			return
		}

		if err := e.evt.Publish(constants.TopicQueueEntityFlush, msg); err != nil {
			logger.Err(err).Msg("error publishing flush event")
		}
	}()

	return nil
}
