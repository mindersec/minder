// Package handlers contains logic for handling webhooks
package handlers

import (
	"context"
	"errors"
	"net/http"
)

// ErrCantParse is returned when a Handler cannot understand the webhook
var ErrCantParse = errors.New("cannot parse webhook")

type WebhookHandler interface {
	Handle(ctx context.Context, r *http.Request) error
}
