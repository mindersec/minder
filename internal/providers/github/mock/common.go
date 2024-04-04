// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/stacklok/minder/internal/providers/github (interfaces: ClientService)
//
// Generated by this command:
//
//	mockgen -package mockgh -destination internal/providers/github/mock/common.go github.com/stacklok/minder/internal/providers/github ClientService
//

// Package mockgh is a generated GoMock package.
package mockgh

import (
	context "context"
	reflect "reflect"

	github "github.com/google/go-github/v60/github"
	gomock "go.uber.org/mock/gomock"
	oauth2 "golang.org/x/oauth2"
)

// MockClientService is a mock of ClientService interface.
type MockClientService struct {
	ctrl     *gomock.Controller
	recorder *MockClientServiceMockRecorder
}

// MockClientServiceMockRecorder is the mock recorder for MockClientService.
type MockClientServiceMockRecorder struct {
	mock *MockClientService
}

// NewMockClientService creates a new mock instance.
func NewMockClientService(ctrl *gomock.Controller) *MockClientService {
	mock := &MockClientService{ctrl: ctrl}
	mock.recorder = &MockClientServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClientService) EXPECT() *MockClientServiceMockRecorder {
	return m.recorder
}

// DeleteInstallation mocks base method.
func (m *MockClientService) DeleteInstallation(arg0 context.Context, arg1 int64, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteInstallation", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteInstallation indicates an expected call of DeleteInstallation.
func (mr *MockClientServiceMockRecorder) DeleteInstallation(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteInstallation", reflect.TypeOf((*MockClientService)(nil).DeleteInstallation), arg0, arg1, arg2)
}

// GetInstallation mocks base method.
func (m *MockClientService) GetInstallation(arg0 context.Context, arg1 int64, arg2 string) (*github.Installation, *github.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInstallation", arg0, arg1, arg2)
	ret0, _ := ret[0].(*github.Installation)
	ret1, _ := ret[1].(*github.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetInstallation indicates an expected call of GetInstallation.
func (mr *MockClientServiceMockRecorder) GetInstallation(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInstallation", reflect.TypeOf((*MockClientService)(nil).GetInstallation), arg0, arg1, arg2)
}

// GetUserIdFromToken mocks base method.
func (m *MockClientService) GetUserIdFromToken(arg0 context.Context, arg1 *oauth2.Token) (*int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserIdFromToken", arg0, arg1)
	ret0, _ := ret[0].(*int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserIdFromToken indicates an expected call of GetUserIdFromToken.
func (mr *MockClientServiceMockRecorder) GetUserIdFromToken(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserIdFromToken", reflect.TypeOf((*MockClientService)(nil).GetUserIdFromToken), arg0, arg1)
}

// ListUserInstallations mocks base method.
func (m *MockClientService) ListUserInstallations(arg0 context.Context, arg1 *oauth2.Token) ([]*github.Installation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUserInstallations", arg0, arg1)
	ret0, _ := ret[0].([]*github.Installation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUserInstallations indicates an expected call of ListUserInstallations.
func (mr *MockClientServiceMockRecorder) ListUserInstallations(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUserInstallations", reflect.TypeOf((*MockClientService)(nil).ListUserInstallations), arg0, arg1)
}
