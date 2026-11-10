// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating RepositoryService
// fixtures and is used in various parts of the code. For testing use
// only.
//
//nolint:all
package fixtures

import (
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/entities/models"
	"github.com/mindersec/minder/internal/entities/properties/service"
	mockSvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
)

type (
	MockPropertyServiceBuilder = func(*gomock.Controller) *mockSvc.MockPropertiesService
	MockPropertyServiceOption  = func(*mockSvc.MockPropertiesService)
)

func NewMockPropertiesService(
	funcs ...MockPropertyServiceOption,
) MockPropertyServiceBuilder {
	return func(ctrl *gomock.Controller) *mockSvc.MockPropertiesService {
		mockPropSvc := mockSvc.NewMockPropertiesService(ctrl)

		for _, fn := range funcs {
			fn(mockPropSvc)
		}

		return mockPropSvc
	}
}

func WithSuccessfulEntityByUpstreamHint(
	ewp *models.EntityWithProperties,
	hint service.ByUpstreamHint,
) MockPropertyServiceOption {
	return func(mockPropSvc *mockSvc.MockPropertiesService) {
		mockPropSvc.EXPECT().EntityWithPropertiesByUpstreamHint(gomock.Any(), ewp.Entity.Type, gomock.Any(), hint, gomock.Any()).
			Return(ewp, nil)
	}
}

func WithFailedEntityByUpstreamHint(
	err error,
) MockPropertyServiceOption {
	return func(mockPropSvc *mockSvc.MockPropertiesService) {
		mockPropSvc.EXPECT().
			EntityWithPropertiesByUpstreamHint(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, err)
	}
}

func WithSuccessfulEntityWithPropertiesByID(
	entityID uuid.UUID,
	ewp *models.EntityWithProperties,
) MockPropertyServiceOption {
	return func(mockPropSvc *mockSvc.MockPropertiesService) {
		mockPropSvc.EXPECT().EntityWithPropertiesByID(gomock.Any(), entityID, gomock.Any()).
			Return(ewp, nil)
	}
}

func WithSuccessfulRetrieveAllPropertiesForEntity() MockPropertyServiceOption {
	return func(mockPropSvc *mockSvc.MockPropertiesService) {
		mockPropSvc.EXPECT().
			RetrieveAllPropertiesForEntity(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)
	}
}

func WithFailedRetrieveAllPropertiesForEntity(
	err error,
) MockPropertyServiceOption {
	return func(mockPropSvc *mockSvc.MockPropertiesService) {
		mockPropSvc.EXPECT().
			RetrieveAllPropertiesForEntity(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(err)
	}
}

func WithSuccessfulEntityWithPropertiesAsProto(
	message protoreflect.ProtoMessage,
) MockPropertyServiceOption {
	return func(mockPropSvc *mockSvc.MockPropertiesService) {
		mockPropSvc.EXPECT().
			EntityWithPropertiesAsProto(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(message, nil)
	}
}

func WithFailedEntityWithPropertiesAsProto(
	err error,
) MockPropertyServiceOption {
	return func(mockPropSvc *mockSvc.MockPropertiesService) {
		mockPropSvc.EXPECT().
			EntityWithPropertiesAsProto(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, err)
	}
}

func WithSuccessfulSaveAllProperties() MockPropertyServiceOption {
	return func(mockPropSvc *mockSvc.MockPropertiesService) {
		mockPropSvc.EXPECT().
			SaveAllProperties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)
	}
}
