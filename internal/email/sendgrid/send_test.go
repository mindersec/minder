// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package sendgrid

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/stretchr/testify/require"
)

// TODO: add a test which uses New, replace client, then Register and sends events

func TestSendEmail(t *testing.T) {
	t.Parallel()

	keyFilePath := filepath.Join(t.TempDir(), "keyfile")
	if err := os.WriteFile(keyFilePath, []byte("SECRET"), 0600); err != nil {
		t.Errorf("Failed to write %s: %s", keyFilePath, err)
	}

	tests := []struct {
		name           string
		to             string
		subject        string
		bodyHTML       string
		bodyText       string
		expectedError  bool
		expectedStatus int
		expectedMsg    []mail.SGMailV3
	}{
		{
			name:           "Successful send",
			to:             "recipient@example.com",
			subject:        "Test Subject",
			bodyHTML:       "<p>Test Body</p>",
			bodyText:       "Test Body",
			expectedStatus: 202,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &fakeSendGrid{}
			sender, err := New(server.SendGrid{
				Sender:     "friendly <hello@example.com>",
				ApiKeyFile: keyFilePath,
			})
			sender.client = mockClient

			ctx := context.Background()

			err = sender.sendEmail(ctx, tt.to, tt.subject, tt.bodyHTML, tt.bodyText)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if diff := cmp.Diff(mockClient.stored, tt.expectedMsg); diff != "" {
					t.Errorf("Unexpected messages (-want +got):\n%s", diff)
				}
			}
		})
	}
}

type fakeSendGrid struct {
	stored []mail.SGMailV3
}

func (f *fakeSendGrid) SendWithContext(ctx context.Context, msg *mail.SGMailV3) (*rest.Response, error) {
	f.stored = append(f.stored, *msg)
	// TODO: add failure cases
	return &rest.Response{
		StatusCode: 202,
	}, nil
}
