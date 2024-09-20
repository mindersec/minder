// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance cf.With the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package fixtures contains code for creating RepositoryService
// fixtures and is used in various parts of the code. For testing use
// only.
//
//nolint:all
package fixtures

import (
	"github.com/google/uuid"
	"github.com/stacklok/minder/internal/entities/properties"
	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties/service"
	mockSvc "github.com/stacklok/minder/internal/entities/properties/service/mock"
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

func WithSuccessfulRetrieveAllProperties(
	expProject uuid.UUID,
	expProvider uuid.UUID,
	expEntityType minder.Entity,
	retProps *properties.Properties,
) MockPropertyServiceOption {
	return func(mockPropSvc *mockSvc.MockPropertiesService) {
		mockPropSvc.EXPECT().
			RetrieveAllProperties(
				gomock.Any(), gomock.Any(),
				expProject, expProvider,
				gomock.Any(),
				expEntityType,
				gomock.Any()).
			Return(retProps, nil)
	}
}
