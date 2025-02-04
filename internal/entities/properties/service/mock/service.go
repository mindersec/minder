// Code generated by MockGen. DO NOT EDIT.
// Source: ./service.go
//
// Generated by this command:
//
//	mockgen -package mock_service -destination=./mock/service.go -source=./service.go
//

// Package mock_service is a generated GoMock package.
package mock_service

import (
	context "context"
	reflect "reflect"

	uuid "github.com/google/uuid"
	models "github.com/mindersec/minder/internal/entities/models"
	properties "github.com/mindersec/minder/pkg/entities/properties"
	service "github.com/mindersec/minder/internal/entities/properties/service"
	manager "github.com/mindersec/minder/internal/providers/manager"
	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v10 "github.com/mindersec/minder/pkg/providers/v1"
	gomock "go.uber.org/mock/gomock"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

// MockPropertiesService is a mock of PropertiesService interface.
type MockPropertiesService struct {
	ctrl     *gomock.Controller
	recorder *MockPropertiesServiceMockRecorder
	isgomock struct{}
}

// MockPropertiesServiceMockRecorder is the mock recorder for MockPropertiesService.
type MockPropertiesServiceMockRecorder struct {
	mock *MockPropertiesService
}

// NewMockPropertiesService creates a new mock instance.
func NewMockPropertiesService(ctrl *gomock.Controller) *MockPropertiesService {
	mock := &MockPropertiesService{ctrl: ctrl}
	mock.recorder = &MockPropertiesServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPropertiesService) EXPECT() *MockPropertiesServiceMockRecorder {
	return m.recorder
}

// EntityWithPropertiesAsProto mocks base method.
func (m *MockPropertiesService) EntityWithPropertiesAsProto(ctx context.Context, ewp *models.EntityWithProperties, provMgr manager.ProviderManager) (protoreflect.ProtoMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EntityWithPropertiesAsProto", ctx, ewp, provMgr)
	ret0, _ := ret[0].(protoreflect.ProtoMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EntityWithPropertiesAsProto indicates an expected call of EntityWithPropertiesAsProto.
func (mr *MockPropertiesServiceMockRecorder) EntityWithPropertiesAsProto(ctx, ewp, provMgr any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EntityWithPropertiesAsProto", reflect.TypeOf((*MockPropertiesService)(nil).EntityWithPropertiesAsProto), ctx, ewp, provMgr)
}

// EntityWithPropertiesByID mocks base method.
func (m *MockPropertiesService) EntityWithPropertiesByID(ctx context.Context, entityID uuid.UUID, opts *service.CallOptions) (*models.EntityWithProperties, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EntityWithPropertiesByID", ctx, entityID, opts)
	ret0, _ := ret[0].(*models.EntityWithProperties)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EntityWithPropertiesByID indicates an expected call of EntityWithPropertiesByID.
func (mr *MockPropertiesServiceMockRecorder) EntityWithPropertiesByID(ctx, entityID, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EntityWithPropertiesByID", reflect.TypeOf((*MockPropertiesService)(nil).EntityWithPropertiesByID), ctx, entityID, opts)
}

// EntityWithPropertiesByUpstreamHint mocks base method.
func (m *MockPropertiesService) EntityWithPropertiesByUpstreamHint(ctx context.Context, entType v1.Entity, getByProps *properties.Properties, hint service.ByUpstreamHint, opts *service.CallOptions) (*models.EntityWithProperties, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EntityWithPropertiesByUpstreamHint", ctx, entType, getByProps, hint, opts)
	ret0, _ := ret[0].(*models.EntityWithProperties)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EntityWithPropertiesByUpstreamHint indicates an expected call of EntityWithPropertiesByUpstreamHint.
func (mr *MockPropertiesServiceMockRecorder) EntityWithPropertiesByUpstreamHint(ctx, entType, getByProps, hint, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EntityWithPropertiesByUpstreamHint", reflect.TypeOf((*MockPropertiesService)(nil).EntityWithPropertiesByUpstreamHint), ctx, entType, getByProps, hint, opts)
}

// ReplaceAllProperties mocks base method.
func (m *MockPropertiesService) ReplaceAllProperties(ctx context.Context, entityID uuid.UUID, props *properties.Properties, opts *service.CallOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReplaceAllProperties", ctx, entityID, props, opts)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReplaceAllProperties indicates an expected call of ReplaceAllProperties.
func (mr *MockPropertiesServiceMockRecorder) ReplaceAllProperties(ctx, entityID, props, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReplaceAllProperties", reflect.TypeOf((*MockPropertiesService)(nil).ReplaceAllProperties), ctx, entityID, props, opts)
}

// ReplaceProperty mocks base method.
func (m *MockPropertiesService) ReplaceProperty(ctx context.Context, entityID uuid.UUID, key string, prop *properties.Property, opts *service.CallOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReplaceProperty", ctx, entityID, key, prop, opts)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReplaceProperty indicates an expected call of ReplaceProperty.
func (mr *MockPropertiesServiceMockRecorder) ReplaceProperty(ctx, entityID, key, prop, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReplaceProperty", reflect.TypeOf((*MockPropertiesService)(nil).ReplaceProperty), ctx, entityID, key, prop, opts)
}

// RetrieveAllProperties mocks base method.
func (m *MockPropertiesService) RetrieveAllProperties(ctx context.Context, provider v10.Provider, projectId, providerID uuid.UUID, lookupProperties *properties.Properties, entType v1.Entity, opts *service.ReadOptions) (*properties.Properties, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RetrieveAllProperties", ctx, provider, projectId, providerID, lookupProperties, entType, opts)
	ret0, _ := ret[0].(*properties.Properties)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RetrieveAllProperties indicates an expected call of RetrieveAllProperties.
func (mr *MockPropertiesServiceMockRecorder) RetrieveAllProperties(ctx, provider, projectId, providerID, lookupProperties, entType, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RetrieveAllProperties", reflect.TypeOf((*MockPropertiesService)(nil).RetrieveAllProperties), ctx, provider, projectId, providerID, lookupProperties, entType, opts)
}

// RetrieveAllPropertiesForEntity mocks base method.
func (m *MockPropertiesService) RetrieveAllPropertiesForEntity(ctx context.Context, efp *models.EntityWithProperties, provMan manager.ProviderManager, opts *service.ReadOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RetrieveAllPropertiesForEntity", ctx, efp, provMan, opts)
	ret0, _ := ret[0].(error)
	return ret0
}

// RetrieveAllPropertiesForEntity indicates an expected call of RetrieveAllPropertiesForEntity.
func (mr *MockPropertiesServiceMockRecorder) RetrieveAllPropertiesForEntity(ctx, efp, provMan, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RetrieveAllPropertiesForEntity", reflect.TypeOf((*MockPropertiesService)(nil).RetrieveAllPropertiesForEntity), ctx, efp, provMan, opts)
}

// SaveAllProperties mocks base method.
func (m *MockPropertiesService) SaveAllProperties(ctx context.Context, entityID uuid.UUID, props *properties.Properties, opts *service.CallOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveAllProperties", ctx, entityID, props, opts)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveAllProperties indicates an expected call of SaveAllProperties.
func (mr *MockPropertiesServiceMockRecorder) SaveAllProperties(ctx, entityID, props, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveAllProperties", reflect.TypeOf((*MockPropertiesService)(nil).SaveAllProperties), ctx, entityID, props, opts)
}
