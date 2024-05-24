//
// Copyright 2024 Stacklok, Inc
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

package installations

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/db"
	mockprovsvc "github.com/stacklok/minder/internal/providers/github/service/mock"
)

func testNewInstallationManager(t *testing.T, mockSvc *mockprovsvc.MockGitHubProviderService) *InstallationManager {
	t.Helper()

	return NewInstallationManager(mockSvc)
}

func TestHandleProviderInstanceRemovedMessage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := mockprovsvc.NewMockGitHubProviderService(ctrl)
	im := testNewInstallationManager(t, mockSvc)

	installationID := 123
	payload := []byte(`{"installation_id": 123}`)
	msg := message.NewMessage(uuid.New().String(), nil)
	iiw := NewInstallationInfoWrapper().
		WithProviderClass(db.ProviderClassGithubApp).
		WithPayload(payload)
	err := iiw.ToMessage(msg)
	require.Nil(t, err)

	mockSvc.EXPECT().
		DeleteGitHubAppInstallation(gomock.Any(), int64(installationID)).
		Return(nil)

	err = im.handleProviderInstallationEvent(msg)
	require.NoError(t, err)
}

func TestHandleUnknownEvent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := mockprovsvc.NewMockGitHubProviderService(ctrl)
	im := testNewInstallationManager(t, mockSvc)

	msg := message.NewMessage(uuid.New().String(), nil)
	msg.Metadata.Set(InstallationEventKey, "unknown")

	// no error but note that we don't have to EXPECT any mock
	// which means the handler just ignores the unknown event

	err := im.handleProviderInstallationEvent(msg)
	require.NoError(t, err)
}

func TestInstallationEntityWrapper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		providerClass db.ProviderClass
		payload       []byte
		error         bool
		message       func(*testing.T, []byte, *message.Message)
	}{
		{
			name:          "happy path",
			providerClass: db.ProviderClassGithubApp,
			payload:       []byte("some payload"),
			//nolint:thelper
			message: func(t *testing.T, payload []byte, msg *message.Message) {
				require.Equal(t, string(ProviderInstanceRemovedEvent), msg.Metadata.Get(InstallationEventKey))
				require.Equal(t, string(db.ProviderClassGithubApp), msg.Metadata.Get(ClassKey))
				require.Equal(t, payload, []byte(msg.Payload))
			},
		},
		{
			name:    "no provider class",
			payload: []byte("some payload"),
			error:   true,
		},
		{
			name:          "no payload",
			providerClass: db.ProviderClassGithubApp,
			error:         true,
		},
		{
			name:          "empty payload",
			providerClass: db.ProviderClassGithubApp,
			payload:       []byte{},
			error:         true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			iiw := NewInstallationInfoWrapper().
				WithProviderClass(tt.providerClass).
				WithPayload(tt.payload)

			m := message.NewMessage(uuid.New().String(), nil)
			err := iiw.ToMessage(m)

			if tt.error {
				require.NotNil(t, err)
				return
			}
			if tt.message != nil {
				tt.message(t, tt.payload, m)
			}
		})
	}
}
