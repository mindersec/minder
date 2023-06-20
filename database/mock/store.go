// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/stacklok/mediator/pkg/db (interfaces: Store)

// Package mockdb is a generated GoMock package.
package mockdb

import (
	context "context"
	sql "database/sql"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	db "github.com/stacklok/mediator/pkg/db"
)

// MockStore is a mock of Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// AddUserGroup mocks base method.
func (m *MockStore) AddUserGroup(arg0 context.Context, arg1 db.AddUserGroupParams) (db.UserGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddUserGroup", arg0, arg1)
	ret0, _ := ret[0].(db.UserGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddUserGroup indicates an expected call of AddUserGroup.
func (mr *MockStoreMockRecorder) AddUserGroup(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddUserGroup", reflect.TypeOf((*MockStore)(nil).AddUserGroup), arg0, arg1)
}

// AddUserRole mocks base method.
func (m *MockStore) AddUserRole(arg0 context.Context, arg1 db.AddUserRoleParams) (db.UserRole, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddUserRole", arg0, arg1)
	ret0, _ := ret[0].(db.UserRole)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddUserRole indicates an expected call of AddUserRole.
func (mr *MockStoreMockRecorder) AddUserRole(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddUserRole", reflect.TypeOf((*MockStore)(nil).AddUserRole), arg0, arg1)
}

// CheckHealth mocks base method.
func (m *MockStore) CheckHealth() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckHealth")
	ret0, _ := ret[0].(error)
	return ret0
}

// CheckHealth indicates an expected call of CheckHealth.
func (mr *MockStoreMockRecorder) CheckHealth() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckHealth", reflect.TypeOf((*MockStore)(nil).CheckHealth))
}

// CleanTokenIat mocks base method.
func (m *MockStore) CleanTokenIat(arg0 context.Context, arg1 int32) (db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CleanTokenIat", arg0, arg1)
	ret0, _ := ret[0].(db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CleanTokenIat indicates an expected call of CleanTokenIat.
func (mr *MockStoreMockRecorder) CleanTokenIat(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CleanTokenIat", reflect.TypeOf((*MockStore)(nil).CleanTokenIat), arg0, arg1)
}

// CreateAccessToken mocks base method.
func (m *MockStore) CreateAccessToken(arg0 context.Context, arg1 db.CreateAccessTokenParams) (db.ProviderAccessToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateAccessToken", arg0, arg1)
	ret0, _ := ret[0].(db.ProviderAccessToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateAccessToken indicates an expected call of CreateAccessToken.
func (mr *MockStoreMockRecorder) CreateAccessToken(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAccessToken", reflect.TypeOf((*MockStore)(nil).CreateAccessToken), arg0, arg1)
}

// CreateGroup mocks base method.
func (m *MockStore) CreateGroup(arg0 context.Context, arg1 db.CreateGroupParams) (db.Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateGroup", arg0, arg1)
	ret0, _ := ret[0].(db.Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateGroup indicates an expected call of CreateGroup.
func (mr *MockStoreMockRecorder) CreateGroup(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateGroup", reflect.TypeOf((*MockStore)(nil).CreateGroup), arg0, arg1)
}

// CreateOrganization mocks base method.
func (m *MockStore) CreateOrganization(arg0 context.Context, arg1 db.CreateOrganizationParams) (db.Organization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrganization", arg0, arg1)
	ret0, _ := ret[0].(db.Organization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateOrganization indicates an expected call of CreateOrganization.
func (mr *MockStoreMockRecorder) CreateOrganization(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrganization", reflect.TypeOf((*MockStore)(nil).CreateOrganization), arg0, arg1)
}

// CreateRepository mocks base method.
func (m *MockStore) CreateRepository(arg0 context.Context, arg1 db.CreateRepositoryParams) (db.Repository, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRepository", arg0, arg1)
	ret0, _ := ret[0].(db.Repository)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRepository indicates an expected call of CreateRepository.
func (mr *MockStoreMockRecorder) CreateRepository(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRepository", reflect.TypeOf((*MockStore)(nil).CreateRepository), arg0, arg1)
}

// CreateRole mocks base method.
func (m *MockStore) CreateRole(arg0 context.Context, arg1 db.CreateRoleParams) (db.Role, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateRole", arg0, arg1)
	ret0, _ := ret[0].(db.Role)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateRole indicates an expected call of CreateRole.
func (mr *MockStoreMockRecorder) CreateRole(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateRole", reflect.TypeOf((*MockStore)(nil).CreateRole), arg0, arg1)
}

// CreateSessionState mocks base method.
func (m *MockStore) CreateSessionState(arg0 context.Context, arg1 db.CreateSessionStateParams) (db.SessionStore, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSessionState", arg0, arg1)
	ret0, _ := ret[0].(db.SessionStore)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSessionState indicates an expected call of CreateSessionState.
func (mr *MockStoreMockRecorder) CreateSessionState(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSessionState", reflect.TypeOf((*MockStore)(nil).CreateSessionState), arg0, arg1)
}

// CreateUser mocks base method.
func (m *MockStore) CreateUser(arg0 context.Context, arg1 db.CreateUserParams) (db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", arg0, arg1)
	ret0, _ := ret[0].(db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockStoreMockRecorder) CreateUser(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockStore)(nil).CreateUser), arg0, arg1)
}

// DeleteAccessToken mocks base method.
func (m *MockStore) DeleteAccessToken(arg0 context.Context, arg1 db.DeleteAccessTokenParams) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAccessToken", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAccessToken indicates an expected call of DeleteAccessToken.
func (mr *MockStoreMockRecorder) DeleteAccessToken(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAccessToken", reflect.TypeOf((*MockStore)(nil).DeleteAccessToken), arg0, arg1)
}

// DeleteExpiredSessionStates mocks base method.
func (m *MockStore) DeleteExpiredSessionStates(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteExpiredSessionStates", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteExpiredSessionStates indicates an expected call of DeleteExpiredSessionStates.
func (mr *MockStoreMockRecorder) DeleteExpiredSessionStates(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteExpiredSessionStates", reflect.TypeOf((*MockStore)(nil).DeleteExpiredSessionStates), arg0)
}

// DeleteGroup mocks base method.
func (m *MockStore) DeleteGroup(arg0 context.Context, arg1 int32) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteGroup", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteGroup indicates an expected call of DeleteGroup.
func (mr *MockStoreMockRecorder) DeleteGroup(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteGroup", reflect.TypeOf((*MockStore)(nil).DeleteGroup), arg0, arg1)
}

// DeleteOrganization mocks base method.
func (m *MockStore) DeleteOrganization(arg0 context.Context, arg1 int32) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteOrganization", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteOrganization indicates an expected call of DeleteOrganization.
func (mr *MockStoreMockRecorder) DeleteOrganization(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteOrganization", reflect.TypeOf((*MockStore)(nil).DeleteOrganization), arg0, arg1)
}

// DeleteRepository mocks base method.
func (m *MockStore) DeleteRepository(arg0 context.Context, arg1 int32) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRepository", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRepository indicates an expected call of DeleteRepository.
func (mr *MockStoreMockRecorder) DeleteRepository(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRepository", reflect.TypeOf((*MockStore)(nil).DeleteRepository), arg0, arg1)
}

// DeleteRole mocks base method.
func (m *MockStore) DeleteRole(arg0 context.Context, arg1 int32) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteRole", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteRole indicates an expected call of DeleteRole.
func (mr *MockStoreMockRecorder) DeleteRole(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteRole", reflect.TypeOf((*MockStore)(nil).DeleteRole), arg0, arg1)
}

// DeleteSessionState mocks base method.
func (m *MockStore) DeleteSessionState(arg0 context.Context, arg1 int32) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSessionState", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSessionState indicates an expected call of DeleteSessionState.
func (mr *MockStoreMockRecorder) DeleteSessionState(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSessionState", reflect.TypeOf((*MockStore)(nil).DeleteSessionState), arg0, arg1)
}

// DeleteSessionStateByGroupID mocks base method.
func (m *MockStore) DeleteSessionStateByGroupID(arg0 context.Context, arg1 sql.NullInt32) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSessionStateByGroupID", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSessionStateByGroupID indicates an expected call of DeleteSessionStateByGroupID.
func (mr *MockStoreMockRecorder) DeleteSessionStateByGroupID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSessionStateByGroupID", reflect.TypeOf((*MockStore)(nil).DeleteSessionStateByGroupID), arg0, arg1)
}

// DeleteUser mocks base method.
func (m *MockStore) DeleteUser(arg0 context.Context, arg1 int32) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUser", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteUser indicates an expected call of DeleteUser.
func (mr *MockStoreMockRecorder) DeleteUser(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUser", reflect.TypeOf((*MockStore)(nil).DeleteUser), arg0, arg1)
}

// GetAccessTokenByGroupID mocks base method.
func (m *MockStore) GetAccessTokenByGroupID(arg0 context.Context, arg1 db.GetAccessTokenByGroupIDParams) (db.ProviderAccessToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccessTokenByGroupID", arg0, arg1)
	ret0, _ := ret[0].(db.ProviderAccessToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccessTokenByGroupID indicates an expected call of GetAccessTokenByGroupID.
func (mr *MockStoreMockRecorder) GetAccessTokenByGroupID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccessTokenByGroupID", reflect.TypeOf((*MockStore)(nil).GetAccessTokenByGroupID), arg0, arg1)
}

// GetAccessTokenByProvider mocks base method.
func (m *MockStore) GetAccessTokenByProvider(arg0 context.Context, arg1 string) ([]db.ProviderAccessToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccessTokenByProvider", arg0, arg1)
	ret0, _ := ret[0].([]db.ProviderAccessToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccessTokenByProvider indicates an expected call of GetAccessTokenByProvider.
func (mr *MockStoreMockRecorder) GetAccessTokenByProvider(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccessTokenByProvider", reflect.TypeOf((*MockStore)(nil).GetAccessTokenByProvider), arg0, arg1)
}

// GetAccessTokenSinceDate mocks base method.
func (m *MockStore) GetAccessTokenSinceDate(arg0 context.Context, arg1 db.GetAccessTokenSinceDateParams) (db.ProviderAccessToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAccessTokenSinceDate", arg0, arg1)
	ret0, _ := ret[0].(db.ProviderAccessToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAccessTokenSinceDate indicates an expected call of GetAccessTokenSinceDate.
func (mr *MockStoreMockRecorder) GetAccessTokenSinceDate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAccessTokenSinceDate", reflect.TypeOf((*MockStore)(nil).GetAccessTokenSinceDate), arg0, arg1)
}

// GetGroupByID mocks base method.
func (m *MockStore) GetGroupByID(arg0 context.Context, arg1 int32) (db.Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroupByID", arg0, arg1)
	ret0, _ := ret[0].(db.Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGroupByID indicates an expected call of GetGroupByID.
func (mr *MockStoreMockRecorder) GetGroupByID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroupByID", reflect.TypeOf((*MockStore)(nil).GetGroupByID), arg0, arg1)
}

// GetGroupByName mocks base method.
func (m *MockStore) GetGroupByName(arg0 context.Context, arg1 string) (db.Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroupByName", arg0, arg1)
	ret0, _ := ret[0].(db.Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGroupByName indicates an expected call of GetGroupByName.
func (mr *MockStoreMockRecorder) GetGroupByName(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroupByName", reflect.TypeOf((*MockStore)(nil).GetGroupByName), arg0, arg1)
}

// GetGroupIDPortBySessionState mocks base method.
func (m *MockStore) GetGroupIDPortBySessionState(arg0 context.Context, arg1 string) (db.GetGroupIDPortBySessionStateRow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroupIDPortBySessionState", arg0, arg1)
	ret0, _ := ret[0].(db.GetGroupIDPortBySessionStateRow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetGroupIDPortBySessionState indicates an expected call of GetGroupIDPortBySessionState.
func (mr *MockStoreMockRecorder) GetGroupIDPortBySessionState(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroupIDPortBySessionState", reflect.TypeOf((*MockStore)(nil).GetGroupIDPortBySessionState), arg0, arg1)
}

// GetOrganization mocks base method.
func (m *MockStore) GetOrganization(arg0 context.Context, arg1 int32) (db.Organization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOrganization", arg0, arg1)
	ret0, _ := ret[0].(db.Organization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOrganization indicates an expected call of GetOrganization.
func (mr *MockStoreMockRecorder) GetOrganization(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOrganization", reflect.TypeOf((*MockStore)(nil).GetOrganization), arg0, arg1)
}

// GetOrganizationByName mocks base method.
func (m *MockStore) GetOrganizationByName(arg0 context.Context, arg1 string) (db.Organization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOrganizationByName", arg0, arg1)
	ret0, _ := ret[0].(db.Organization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOrganizationByName indicates an expected call of GetOrganizationByName.
func (mr *MockStoreMockRecorder) GetOrganizationByName(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOrganizationByName", reflect.TypeOf((*MockStore)(nil).GetOrganizationByName), arg0, arg1)
}

// GetOrganizationForUpdate mocks base method.
func (m *MockStore) GetOrganizationForUpdate(arg0 context.Context, arg1 string) (db.Organization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOrganizationForUpdate", arg0, arg1)
	ret0, _ := ret[0].(db.Organization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOrganizationForUpdate indicates an expected call of GetOrganizationForUpdate.
func (mr *MockStoreMockRecorder) GetOrganizationForUpdate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOrganizationForUpdate", reflect.TypeOf((*MockStore)(nil).GetOrganizationForUpdate), arg0, arg1)
}

// GetRepositoryByID mocks base method.
func (m *MockStore) GetRepositoryByID(arg0 context.Context, arg1 int32) (db.Repository, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRepositoryByID", arg0, arg1)
	ret0, _ := ret[0].(db.Repository)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRepositoryByID indicates an expected call of GetRepositoryByID.
func (mr *MockStoreMockRecorder) GetRepositoryByID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRepositoryByID", reflect.TypeOf((*MockStore)(nil).GetRepositoryByID), arg0, arg1)
}

// GetRepositoryByRepoName mocks base method.
func (m *MockStore) GetRepositoryByRepoName(arg0 context.Context, arg1 string) (db.Repository, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRepositoryByRepoName", arg0, arg1)
	ret0, _ := ret[0].(db.Repository)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRepositoryByRepoName indicates an expected call of GetRepositoryByRepoName.
func (mr *MockStoreMockRecorder) GetRepositoryByRepoName(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRepositoryByRepoName", reflect.TypeOf((*MockStore)(nil).GetRepositoryByRepoName), arg0, arg1)
}

// GetRoleByID mocks base method.
func (m *MockStore) GetRoleByID(arg0 context.Context, arg1 int32) (db.Role, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleByID", arg0, arg1)
	ret0, _ := ret[0].(db.Role)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleByID indicates an expected call of GetRoleByID.
func (mr *MockStoreMockRecorder) GetRoleByID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleByID", reflect.TypeOf((*MockStore)(nil).GetRoleByID), arg0, arg1)
}

// GetRoleByName mocks base method.
func (m *MockStore) GetRoleByName(arg0 context.Context, arg1 db.GetRoleByNameParams) (db.Role, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRoleByName", arg0, arg1)
	ret0, _ := ret[0].(db.Role)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRoleByName indicates an expected call of GetRoleByName.
func (mr *MockStoreMockRecorder) GetRoleByName(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRoleByName", reflect.TypeOf((*MockStore)(nil).GetRoleByName), arg0, arg1)
}

// GetSessionState mocks base method.
func (m *MockStore) GetSessionState(arg0 context.Context, arg1 int32) (db.SessionStore, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSessionState", arg0, arg1)
	ret0, _ := ret[0].(db.SessionStore)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSessionState indicates an expected call of GetSessionState.
func (mr *MockStoreMockRecorder) GetSessionState(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSessionState", reflect.TypeOf((*MockStore)(nil).GetSessionState), arg0, arg1)
}

// GetSessionStateByGroupID mocks base method.
func (m *MockStore) GetSessionStateByGroupID(arg0 context.Context, arg1 sql.NullInt32) (db.SessionStore, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSessionStateByGroupID", arg0, arg1)
	ret0, _ := ret[0].(db.SessionStore)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSessionStateByGroupID indicates an expected call of GetSessionStateByGroupID.
func (mr *MockStoreMockRecorder) GetSessionStateByGroupID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSessionStateByGroupID", reflect.TypeOf((*MockStore)(nil).GetSessionStateByGroupID), arg0, arg1)
}

// GetUserByEmail mocks base method.
func (m *MockStore) GetUserByEmail(arg0 context.Context, arg1 sql.NullString) (db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserByEmail", arg0, arg1)
	ret0, _ := ret[0].(db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserByEmail indicates an expected call of GetUserByEmail.
func (mr *MockStoreMockRecorder) GetUserByEmail(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserByEmail", reflect.TypeOf((*MockStore)(nil).GetUserByEmail), arg0, arg1)
}

// GetUserByID mocks base method.
func (m *MockStore) GetUserByID(arg0 context.Context, arg1 int32) (db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserByID", arg0, arg1)
	ret0, _ := ret[0].(db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserByID indicates an expected call of GetUserByID.
func (mr *MockStoreMockRecorder) GetUserByID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserByID", reflect.TypeOf((*MockStore)(nil).GetUserByID), arg0, arg1)
}

// GetUserByUserName mocks base method.
func (m *MockStore) GetUserByUserName(arg0 context.Context, arg1 string) (db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserByUserName", arg0, arg1)
	ret0, _ := ret[0].(db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserByUserName indicates an expected call of GetUserByUserName.
func (mr *MockStoreMockRecorder) GetUserByUserName(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserByUserName", reflect.TypeOf((*MockStore)(nil).GetUserByUserName), arg0, arg1)
}

// GetUserClaims mocks base method.
func (m *MockStore) GetUserClaims(arg0 context.Context, arg1 int32) (db.GetUserClaimsRow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserClaims", arg0, arg1)
	ret0, _ := ret[0].(db.GetUserClaimsRow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserClaims indicates an expected call of GetUserClaims.
func (mr *MockStoreMockRecorder) GetUserClaims(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserClaims", reflect.TypeOf((*MockStore)(nil).GetUserClaims), arg0, arg1)
}

// ListGroups mocks base method.
func (m *MockStore) ListGroups(arg0 context.Context, arg1 db.ListGroupsParams) ([]db.Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListGroups", arg0, arg1)
	ret0, _ := ret[0].([]db.Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListGroups indicates an expected call of ListGroups.
func (mr *MockStoreMockRecorder) ListGroups(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListGroups", reflect.TypeOf((*MockStore)(nil).ListGroups), arg0, arg1)
}

// ListGroupsByOrganizationID mocks base method.
func (m *MockStore) ListGroupsByOrganizationID(arg0 context.Context, arg1 int32) ([]db.Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListGroupsByOrganizationID", arg0, arg1)
	ret0, _ := ret[0].([]db.Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListGroupsByOrganizationID indicates an expected call of ListGroupsByOrganizationID.
func (mr *MockStoreMockRecorder) ListGroupsByOrganizationID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListGroupsByOrganizationID", reflect.TypeOf((*MockStore)(nil).ListGroupsByOrganizationID), arg0, arg1)
}

// ListOrganizations mocks base method.
func (m *MockStore) ListOrganizations(arg0 context.Context, arg1 db.ListOrganizationsParams) ([]db.Organization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOrganizations", arg0, arg1)
	ret0, _ := ret[0].([]db.Organization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListOrganizations indicates an expected call of ListOrganizations.
func (mr *MockStoreMockRecorder) ListOrganizations(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOrganizations", reflect.TypeOf((*MockStore)(nil).ListOrganizations), arg0, arg1)
}

// ListRepositoriesByGroupID mocks base method.
func (m *MockStore) ListRepositoriesByGroupID(arg0 context.Context, arg1 db.ListRepositoriesByGroupIDParams) ([]db.Repository, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRepositoriesByGroupID", arg0, arg1)
	ret0, _ := ret[0].([]db.Repository)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRepositoriesByGroupID indicates an expected call of ListRepositoriesByGroupID.
func (mr *MockStoreMockRecorder) ListRepositoriesByGroupID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRepositoriesByGroupID", reflect.TypeOf((*MockStore)(nil).ListRepositoriesByGroupID), arg0, arg1)
}

// ListRepositoriesByOwner mocks base method.
func (m *MockStore) ListRepositoriesByOwner(arg0 context.Context, arg1 db.ListRepositoriesByOwnerParams) ([]db.Repository, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRepositoriesByOwner", arg0, arg1)
	ret0, _ := ret[0].([]db.Repository)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRepositoriesByOwner indicates an expected call of ListRepositoriesByOwner.
func (mr *MockStoreMockRecorder) ListRepositoriesByOwner(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRepositoriesByOwner", reflect.TypeOf((*MockStore)(nil).ListRepositoriesByOwner), arg0, arg1)
}

// ListRoles mocks base method.
func (m *MockStore) ListRoles(arg0 context.Context, arg1 db.ListRolesParams) ([]db.Role, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRoles", arg0, arg1)
	ret0, _ := ret[0].([]db.Role)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRoles indicates an expected call of ListRoles.
func (mr *MockStoreMockRecorder) ListRoles(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRoles", reflect.TypeOf((*MockStore)(nil).ListRoles), arg0, arg1)
}

// ListRolesByGroupID mocks base method.
func (m *MockStore) ListRolesByGroupID(arg0 context.Context, arg1 db.ListRolesByGroupIDParams) ([]db.Role, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListRolesByGroupID", arg0, arg1)
	ret0, _ := ret[0].([]db.Role)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListRolesByGroupID indicates an expected call of ListRolesByGroupID.
func (mr *MockStoreMockRecorder) ListRolesByGroupID(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListRolesByGroupID", reflect.TypeOf((*MockStore)(nil).ListRolesByGroupID), arg0, arg1)
}

// ListUsers mocks base method.
func (m *MockStore) ListUsers(arg0 context.Context, arg1 db.ListUsersParams) ([]db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsers", arg0, arg1)
	ret0, _ := ret[0].([]db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsers indicates an expected call of ListUsers.
func (mr *MockStoreMockRecorder) ListUsers(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsers", reflect.TypeOf((*MockStore)(nil).ListUsers), arg0, arg1)
}

// ListUsersByGroup mocks base method.
func (m *MockStore) ListUsersByGroup(arg0 context.Context, arg1 db.ListUsersByGroupParams) ([]db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsersByGroup", arg0, arg1)
	ret0, _ := ret[0].([]db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsersByGroup indicates an expected call of ListUsersByGroup.
func (mr *MockStoreMockRecorder) ListUsersByGroup(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsersByGroup", reflect.TypeOf((*MockStore)(nil).ListUsersByGroup), arg0, arg1)
}

// ListUsersByOrganization mocks base method.
func (m *MockStore) ListUsersByOrganization(arg0 context.Context, arg1 db.ListUsersByOrganizationParams) ([]db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsersByOrganization", arg0, arg1)
	ret0, _ := ret[0].([]db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsersByOrganization indicates an expected call of ListUsersByOrganization.
func (mr *MockStoreMockRecorder) ListUsersByOrganization(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsersByOrganization", reflect.TypeOf((*MockStore)(nil).ListUsersByOrganization), arg0, arg1)
}

// ListUsersByRoleId mocks base method.
func (m *MockStore) ListUsersByRoleId(arg0 context.Context, arg1 int32) ([]int32, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsersByRoleId", arg0, arg1)
	ret0, _ := ret[0].([]int32)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsersByRoleId indicates an expected call of ListUsersByRoleId.
func (mr *MockStoreMockRecorder) ListUsersByRoleId(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsersByRoleId", reflect.TypeOf((*MockStore)(nil).ListUsersByRoleId), arg0, arg1)
}

// RevokeUserToken mocks base method.
func (m *MockStore) RevokeUserToken(arg0 context.Context, arg1 db.RevokeUserTokenParams) (db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RevokeUserToken", arg0, arg1)
	ret0, _ := ret[0].(db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RevokeUserToken indicates an expected call of RevokeUserToken.
func (mr *MockStoreMockRecorder) RevokeUserToken(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RevokeUserToken", reflect.TypeOf((*MockStore)(nil).RevokeUserToken), arg0, arg1)
}

// RevokeUsersTokens mocks base method.
func (m *MockStore) RevokeUsersTokens(arg0 context.Context, arg1 sql.NullTime) (db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RevokeUsersTokens", arg0, arg1)
	ret0, _ := ret[0].(db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RevokeUsersTokens indicates an expected call of RevokeUsersTokens.
func (mr *MockStoreMockRecorder) RevokeUsersTokens(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RevokeUsersTokens", reflect.TypeOf((*MockStore)(nil).RevokeUsersTokens), arg0, arg1)
}

// UpdateAccessToken mocks base method.
func (m *MockStore) UpdateAccessToken(arg0 context.Context, arg1 db.UpdateAccessTokenParams) (db.ProviderAccessToken, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateAccessToken", arg0, arg1)
	ret0, _ := ret[0].(db.ProviderAccessToken)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateAccessToken indicates an expected call of UpdateAccessToken.
func (mr *MockStoreMockRecorder) UpdateAccessToken(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateAccessToken", reflect.TypeOf((*MockStore)(nil).UpdateAccessToken), arg0, arg1)
}

// UpdateGroup mocks base method.
func (m *MockStore) UpdateGroup(arg0 context.Context, arg1 db.UpdateGroupParams) (db.Group, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateGroup", arg0, arg1)
	ret0, _ := ret[0].(db.Group)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateGroup indicates an expected call of UpdateGroup.
func (mr *MockStoreMockRecorder) UpdateGroup(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateGroup", reflect.TypeOf((*MockStore)(nil).UpdateGroup), arg0, arg1)
}

// UpdateOrganization mocks base method.
func (m *MockStore) UpdateOrganization(arg0 context.Context, arg1 db.UpdateOrganizationParams) (db.Organization, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateOrganization", arg0, arg1)
	ret0, _ := ret[0].(db.Organization)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateOrganization indicates an expected call of UpdateOrganization.
func (mr *MockStoreMockRecorder) UpdateOrganization(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateOrganization", reflect.TypeOf((*MockStore)(nil).UpdateOrganization), arg0, arg1)
}

// UpdatePassword mocks base method.
func (m *MockStore) UpdatePassword(arg0 context.Context, arg1 db.UpdatePasswordParams) (db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdatePassword", arg0, arg1)
	ret0, _ := ret[0].(db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdatePassword indicates an expected call of UpdatePassword.
func (mr *MockStoreMockRecorder) UpdatePassword(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdatePassword", reflect.TypeOf((*MockStore)(nil).UpdatePassword), arg0, arg1)
}

// UpdateRepository mocks base method.
func (m *MockStore) UpdateRepository(arg0 context.Context, arg1 db.UpdateRepositoryParams) (db.Repository, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRepository", arg0, arg1)
	ret0, _ := ret[0].(db.Repository)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRepository indicates an expected call of UpdateRepository.
func (mr *MockStoreMockRecorder) UpdateRepository(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRepository", reflect.TypeOf((*MockStore)(nil).UpdateRepository), arg0, arg1)
}

// UpdateRole mocks base method.
func (m *MockStore) UpdateRole(arg0 context.Context, arg1 db.UpdateRoleParams) (db.Role, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRole", arg0, arg1)
	ret0, _ := ret[0].(db.Role)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRole indicates an expected call of UpdateRole.
func (mr *MockStoreMockRecorder) UpdateRole(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRole", reflect.TypeOf((*MockStore)(nil).UpdateRole), arg0, arg1)
}

// UpdateUser mocks base method.
func (m *MockStore) UpdateUser(arg0 context.Context, arg1 db.UpdateUserParams) (db.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUser", arg0, arg1)
	ret0, _ := ret[0].(db.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockStoreMockRecorder) UpdateUser(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockStore)(nil).UpdateUser), arg0, arg1)
}
