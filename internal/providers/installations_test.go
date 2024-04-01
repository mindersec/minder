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

package providers

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/db"
	mockprofsvc "github.com/stacklok/minder/internal/providers/mock"
)

func TestProviderInstanceRemovedMessage(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"installation_id": 123}`)
	msg := message.NewMessage(uuid.New().String(), nil)
	ProviderInstanceRemovedMessage(msg, db.ProviderClassGithubApp, payload)

	require.Equal(t, string(db.ProviderClassGithubApp), msg.Metadata.Get(ClassKey))
	require.Equal(t, string(ProviderInstanceRemovedEvent), msg.Metadata.Get(InstallationEventKey))
}

func testNewInstallationManager(t *testing.T, mockSvc *mockprofsvc.MockProviderService) *InstallationManager {
	t.Helper()

	return NewInstallationManager(mockSvc)
}

func TestHandleProviderInstanceRemovedMessage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := mockprofsvc.NewMockProviderService(ctrl)
	im := testNewInstallationManager(t, mockSvc)

	installationID := 123
	payload := []byte(`{"installation_id": 123}`)
	msg := message.NewMessage(uuid.New().String(), nil)
	ProviderInstanceRemovedMessage(msg, db.ProviderClassGithubApp, payload)

	mockSvc.EXPECT().
		DeleteGitHubAppInstallation(gomock.Any(), int64(installationID)).
		Return(nil)

	err := im.handleProviderInstallationEvent(msg)
	require.NoError(t, err)
}

func TestHandleUnknownEvent(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockSvc := mockprofsvc.NewMockProviderService(ctrl)
	im := testNewInstallationManager(t, mockSvc)

	msg := message.NewMessage(uuid.New().String(), nil)
	msg.Metadata.Set(InstallationEventKey, "unknown")

	// no error but note that we don't have to EXPECT any mock
	// which means the handler just ignores the unknown event

	err := im.handleProviderInstallationEvent(msg)
	require.NoError(t, err)
}
