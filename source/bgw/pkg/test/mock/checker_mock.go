// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/server/filter/openapi/common_check.go

// Package mock_openapi is a generated GoMock package.
package mock

import (
	types "bgw/pkg/common/types"
	reflect "reflect"

	sign "code.bydev.io/fbu/gateway/gway.git/gcore/sign"
	gomock "github.com/golang/mock/gomock"
)

// MockChecker is a mock of Checker interface.
type MockChecker struct {
	ctrl     *gomock.Controller
	recorder *MockCheckerMockRecorder
}

// MockCheckerMockRecorder is the mock recorder for MockChecker.
type MockCheckerMockRecorder struct {
	mock *MockChecker
}

// NewMockChecker creates a new mock instance.
func NewMockChecker(ctrl *gomock.Controller) *MockChecker {
	mock := &MockChecker{ctrl: ctrl}
	mock.recorder = &MockCheckerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChecker) EXPECT() *MockCheckerMockRecorder {
	return m.recorder
}

// GetAPIKey mocks base method.
func (m *MockChecker) GetAPIKey() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAPIKey")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAPIKey indicates an expected call of GetAPIKey.
func (mr *MockCheckerMockRecorder) GetAPIKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAPIKey", reflect.TypeOf((*MockChecker)(nil).GetAPIKey))
}

// GetAPIRecvWindow mocks base method.
func (m *MockChecker) GetAPIRecvWindow() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAPIRecvWindow")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAPIRecvWindow indicates an expected call of GetAPIRecvWindow.
func (mr *MockCheckerMockRecorder) GetAPIRecvWindow() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAPIRecvWindow", reflect.TypeOf((*MockChecker)(nil).GetAPIRecvWindow))
}

// GetAPISign mocks base method.
func (m *MockChecker) GetAPISign() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAPISign")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAPISign indicates an expected call of GetAPISign.
func (mr *MockCheckerMockRecorder) GetAPISign() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAPISign", reflect.TypeOf((*MockChecker)(nil).GetAPISign))
}

// GetAPITimestamp mocks base method.
func (m *MockChecker) GetAPITimestamp() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAPITimestamp")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAPITimestamp indicates an expected call of GetAPITimestamp.
func (mr *MockCheckerMockRecorder) GetAPITimestamp() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAPITimestamp", reflect.TypeOf((*MockChecker)(nil).GetAPITimestamp))
}

// GetClientIP mocks base method.
func (m *MockChecker) GetClientIP() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClientIP")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetClientIP indicates an expected call of GetClientIP.
func (mr *MockCheckerMockRecorder) GetClientIP() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClientIP", reflect.TypeOf((*MockChecker)(nil).GetClientIP))
}

// GetVersion mocks base method.
func (m *MockChecker) GetVersion() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVersion")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetVersion indicates an expected call of GetVersion.
func (mr *MockCheckerMockRecorder) GetVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVersion", reflect.TypeOf((*MockChecker)(nil).GetVersion))
}

// VerifySign mocks base method.
func (m *MockChecker) VerifySign(ctx *types.Ctx, signTyp sign.Type, secret string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VerifySign", ctx, signTyp, secret)
	ret0, _ := ret[0].(error)
	return ret0
}

// VerifySign indicates an expected call of VerifySign.
func (mr *MockCheckerMockRecorder) VerifySign(ctx, signTyp, secret interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifySign", reflect.TypeOf((*MockChecker)(nil).VerifySign), ctx, signTyp, secret)
}
