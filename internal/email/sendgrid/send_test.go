// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package sendgrid

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-cmp/cmp"
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/email"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

func setupKeyFile(t *testing.T) string {
	t.Helper()
	keyFilePath := filepath.Join(t.TempDir(), "keyfile")
	require.NoError(t, os.WriteFile(keyFilePath, []byte("SECRET"), 0600))
	return keyFilePath
}

func newTestSender(t *testing.T, mockClient client) *SendGrid {
	t.Helper()
	sender, err := New(serverconfig.SendGrid{
		Sender:     "friendly <hello@example.com>",
		ApiKeyFile: setupKeyFile(t),
	})
	require.NoError(t, err)
	sender.client = mockClient
	return sender
}

func TestNew(t *testing.T) {
	t.Parallel()

	keyFilePath := setupKeyFile(t)

	tests := []struct {
		name          string
		cfg           serverconfig.SendGrid
		expectedError string
	}{
		{
			name: "valid config",
			cfg: serverconfig.SendGrid{
				Sender:     "friendly <hello@example.com>",
				ApiKeyFile: keyFilePath,
			},
		},
		{
			name:          "empty sender",
			cfg:           serverconfig.SendGrid{ApiKeyFile: keyFilePath},
			expectedError: "sender email address cannot be empty",
		},
		{
			name:          "invalid sender format",
			cfg:           serverconfig.SendGrid{Sender: "not-an-email", ApiKeyFile: keyFilePath},
			expectedError: "incorrect sender email format",
		},
		{
			name:          "empty api key file",
			cfg:           serverconfig.SendGrid{Sender: "friendly <hello@example.com>"},
			expectedError: "SendGrid API key cannot be empty",
		},
		{
			name: "missing api key file",
			cfg: serverconfig.SendGrid{
				Sender:     "friendly <hello@example.com>",
				ApiKeyFile: "/nonexistent/path/keyfile",
			},
			expectedError: "unable to read ApiKeyFile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sg, err := New(tt.cfg)
			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
				require.Nil(t, sg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, sg)
			}
		})
	}
}

func TestSendEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		to            string
		subject       string
		bodyHTML      string
		bodyText      string
		clientStatus  int
		clientErr     error
		expectedError string
		expectedMsg   []mail.SGMailV3
	}{
		{
			name:         "successful send",
			to:           "recipient@example.com",
			subject:      "Test Subject",
			bodyHTML:     "<p>Test Body</p>",
			bodyText:     "Test Body",
			clientStatus: 202,
			expectedMsg: []mail.SGMailV3{
				{
					From:    &mail.Email{Name: "friendly", Address: "hello@example.com"},
					Subject: "Test Subject",
					Personalizations: []*mail.Personalization{
						{
							To:                  []*mail.Email{{Address: "recipient@example.com"}},
							CC:                  []*mail.Email{},
							BCC:                 []*mail.Email{},
							Headers:             map[string]string{},
							Substitutions:       map[string]string{},
							CustomArgs:          map[string]string{},
							DynamicTemplateData: map[string]any{},
							Categories:          []string{},
						},
					},
					Content: []*mail.Content{
						{Type: "text/plain", Value: "Test Body"},
						{Type: "text/html", Value: "<p>Test Body</p>"},
					},
				},
			},
		},
		{
			name:          "client network error",
			to:            "recipient@example.com",
			subject:       "Test Subject",
			bodyHTML:      "<p>Test Body</p>",
			bodyText:      "Test Body",
			clientErr:     errors.New("network failure"),
			expectedError: "network failure",
		},
		{
			name:          "client returns 400",
			to:            "recipient@example.com",
			subject:       "Test Subject",
			bodyHTML:      "<p>Test Body</p>",
			bodyText:      "Test Body",
			clientStatus:  400,
			expectedError: "SendGrid API error: status code 400",
		},
		{
			name:          "client returns 500",
			to:            "recipient@example.com",
			subject:       "Test Subject",
			bodyHTML:      "<p>Test Body</p>",
			bodyText:      "Test Body",
			clientStatus:  500,
			expectedError: "SendGrid API error: status code 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &fakeSendGrid{
				statusCode: tt.clientStatus,
				err:        tt.clientErr,
			}
			sender := newTestSender(t, mockClient)

			err := sender.sendEmail(context.Background(), tt.to, tt.subject, tt.bodyHTML, tt.bodyText)
			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				if diff := cmp.Diff(tt.expectedMsg, mockClient.stored); diff != "" {
					t.Errorf("unexpected messages (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestRegister(t *testing.T) {
	t.Parallel()

	payload := email.MailEventPayload{
		Address:  "recipient@example.com",
		Subject:  "Invite Subject",
		BodyHTML: "<p>You are invited</p>",
		BodyText: "You are invited",
	}
	rawPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	tests := []struct {
		name          string
		payload       []byte
		clientStatus  int
		clientErr     error
		expectedError string
	}{
		{
			name:         "successful registration and send",
			payload:      rawPayload,
			clientStatus: 202,
		},
		{
			name:          "invalid payload",
			payload:       []byte("not-json"),
			clientStatus:  202,
			expectedError: "error unmarshalling invite email event",
		},
		{
			name:          "client error",
			payload:       rawPayload,
			clientErr:     errors.New("smtp down"),
			expectedError: "smtp down",
		},
		{
			name:          "client returns 400",
			payload:       rawPayload,
			clientStatus:  400,
			expectedError: "SendGrid API error: status code 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &fakeSendGrid{
				statusCode: tt.clientStatus,
				err:        tt.clientErr,
			}
			sender := newTestSender(t, mockClient)

			var handlerFn interfaces.Handler
			fakeReg := &fakeRegistrar{
				registerFn: func(_ string, fn interfaces.Handler, _ ...message.HandlerMiddleware) {
					handlerFn = fn
				},
			}
			sender.Register(fakeReg)
			require.NotNil(t, handlerFn, "Register must call reg.Register")

			msg := message.NewMessage("test-id", tt.payload)
			err := handlerFn(msg)
			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

type fakeSendGrid struct {
	mu         sync.Mutex
	stored     []mail.SGMailV3
	statusCode int
	err        error
}

func (f *fakeSendGrid) SendWithContext(_ context.Context, msg *mail.SGMailV3) (*rest.Response, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stored = append(f.stored, *msg)
	return &rest.Response{StatusCode: f.statusCode}, nil
}

func (f *fakeSendGrid) storedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.stored)
}

func (f *fakeSendGrid) storedAt(i int) mail.SGMailV3 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stored[i]
}

type fakeRegistrar struct {
	registerFn func(topic string, fn interfaces.Handler, middleware ...message.HandlerMiddleware)
}

func (f *fakeRegistrar) Register(topic string, fn interfaces.Handler, middleware ...message.HandlerMiddleware) {
	f.registerFn(topic, fn, middleware...)
}

func TestRegisterIntegration(t *testing.T) {
	t.Parallel()

	payload := email.MailEventPayload{
		Address:  "recipient@example.com",
		Subject:  "Invite Subject",
		BodyHTML: "<p>You are invited</p>",
		BodyText: "You are invited",
	}
	rawPayload, err := json.Marshal(payload)
	require.NoError(t, err)

	mockClient := &fakeSendGrid{statusCode: 202}
	sender := newTestSender(t, mockClient)

	evt, err := eventer.New(context.Background(), nil, &serverconfig.EventConfig{
		Driver: "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{
			BlockPublishUntilSubscriberAck: true,
		},
	})
	require.NoError(t, err)

	sender.Register(evt)

	go func() {
		require.NoError(t, evt.Run(context.Background()))
	}()

	<-evt.Running()

	msg := message.NewMessage("test-id", rawPayload)
	err = evt.Publish(email.TopicQueueInviteEmail, msg)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return mockClient.storedCount() == 1
	}, 5*time.Second, 50*time.Millisecond)

	require.Equal(t, "recipient@example.com", mockClient.storedAt(0).Personalizations[0].To[0].Address)
}
