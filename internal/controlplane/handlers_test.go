// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestCheckHealth(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	mockStore.EXPECT().CheckHealth().Return(nil)

	server := Server{
		store: mockStore,
	}
	response, err := server.CheckHealth(context.Background(), &pb.CheckHealthRequest{})
	if err != nil {
		t.Errorf("Error in CheckHealth: %v", err)
	}

	if response.Status != "OK" {
		t.Errorf("Unexpected response from CheckHealth: %v", response)
	}
}
