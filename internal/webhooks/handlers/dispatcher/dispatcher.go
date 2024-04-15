package dispatcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/stacklok/minder/internal/webhooks/handlers"
	"io"
	"net/http"
)

type WebhookHandlerDispatcher struct {
	handlers []handlers.WebhookHandler
}

func (w *WebhookHandlerDispatcher) Dispatch(ctx context.Context, request *http.Request) error {
	// by the end of this function, we will no longer need the original request
	defer request.Body.Close()
	// since http.Request contains a reader, we need to do a deep copy per
	// handler
	clone, err := cloneRequest(ctx, request)
	if err != nil {
		return err
	}

	// naive implementation using a chain of responsibility
	// consider dispatching based on the URL/headers of the request
	for _, handler := range w.handlers {
		err := handler.Handle(ctx, clone())
		if errors.Is(err, handlers.ErrCantParse) {
			// try the next one
			continue
		} else if err != nil {
			return err
		}
		// if we got here - we were successful
		return nil
	}

	// we ran out of handlers to try
	return handlers.ErrCantParse
}

func cloneRequest(ctx context.Context, r *http.Request) (func() *http.Request, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read request body: %w", err)
	}

	return func() *http.Request {
		clonedRequest := r.Clone(ctx)
		// clone does not deep copy the reader, do it by hand
		clonedRequest.Body = io.NopCloser(bytes.NewReader(body))
		return clonedRequest
	}, nil
}
