// Package handlers contains logic for handling webhooks
package handlers

type WebhookHandler interface {
	Handler(body []byte) error
}
