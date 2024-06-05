// Code generated by MockGen. DO NOT EDIT.
// Source: ./service.go
//
// Generated by this command:
//
//	mockgen -package mock_ruletypes -destination=./mock/service.go -source=./service.go
//

// Package mock_ruletypes is a generated GoMock package.
package mock_ruletypes

import (
	context "context"
	reflect "reflect"

	uuid "github.com/google/uuid"
	db "github.com/stacklok/minder/internal/db"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	gomock "go.uber.org/mock/gomock"
)

// MockRuleTypeService is a mock of RuleTypeService interface.
type MockRuleTypeService struct {
	ctrl     *gomock.Controller
	recorder *MockRuleTypeServiceMockRecorder
}

// MockRuleTypeServiceMockRecorder is the mock recorder for MockRuleTypeService.
type MockRuleTypeServiceMockRecorder struct {
	mock *MockRuleTypeService
}

// NewMockRuleTypeService creates a new mock instance.
func NewMockRuleTypeService(ctrl *gomock.Controller) *MockRuleTypeService {
	mock := &MockRuleTypeService{ctrl: ctrl}
	mock.recorder = &MockRuleTypeServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRuleTypeService) EXPECT() *MockRuleTypeServiceMockRecorder {
	return m.recorder
}

// CreateRuleType mocks base method.
func (m *MockRuleTypeService) CreateRuleType(ctx context.Context, projectID, subscriptionID uuid.UUID, ruleType *v1.RuleType, qtx db.Querier) (*v1.RuleType, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRuleType", ctx, projectID, subscriptionID, ruleType, qtx)
	ret0, _ := ret[0].(*v1.RuleType)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRuleType indicates an expected call of CreateRuleType.
func (mr *MockRuleTypeServiceMockRecorder) CreateRuleType(ctx, projectID, subscriptionID, ruleType, qtx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRuleType", reflect.TypeOf((*MockRuleTypeService)(nil).CreateRuleType), ctx, projectID, subscriptionID, ruleType, qtx)
}

// UpdateRuleType mocks base method.
func (m *MockRuleTypeService) UpdateRuleType(ctx context.Context, projectID, subscriptionID uuid.UUID, ruleType *v1.RuleType, qtx db.Querier) (*v1.RuleType, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRuleType", ctx, projectID, subscriptionID, ruleType, qtx)
	ret0, _ := ret[0].(*v1.RuleType)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRuleType indicates an expected call of UpdateRuleType.
func (mr *MockRuleTypeServiceMockRecorder) UpdateRuleType(ctx, projectID, subscriptionID, ruleType, qtx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRuleType", reflect.TypeOf((*MockRuleTypeService)(nil).UpdateRuleType), ctx, projectID, subscriptionID, ruleType, qtx)
}

// UpsertRuleType mocks base method.
func (m *MockRuleTypeService) UpsertRuleType(ctx context.Context, projectID, subscriptionID uuid.UUID, ruleType *v1.RuleType, qtx db.Querier) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpsertRuleType", ctx, projectID, subscriptionID, ruleType, qtx)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpsertRuleType indicates an expected call of UpsertRuleType.
func (mr *MockRuleTypeServiceMockRecorder) UpsertRuleType(ctx, projectID, subscriptionID, ruleType, qtx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpsertRuleType", reflect.TypeOf((*MockRuleTypeService)(nil).UpsertRuleType), ctx, projectID, subscriptionID, ruleType, qtx)
}