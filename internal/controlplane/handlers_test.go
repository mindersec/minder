// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
