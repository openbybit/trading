// Code generated by MockGen. DO NOT EDIT.
// Source: conn.go

// Package pool is a generated GoMock package.
package pool

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	grpc "google.golang.org/grpc"
)

// MockConn is a mock of Conn interface.
type MockConn struct {
	ctrl     *gomock.Controller
	recorder *MockConnMockRecorder
}

// MockConnMockRecorder is the mock recorder for MockConn.
type MockConnMockRecorder struct {
	mock *MockConn
}

// NewMockConn creates a new mock instance.
func NewMockConn(ctrl *gomock.Controller) *MockConn {
	mock := &MockConn{ctrl: ctrl}
	mock.recorder = &MockConnMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConn) EXPECT() *MockConnMockRecorder {
	return m.recorder
}

// Client mocks base method.
func (m *MockConn) Client() *grpc.ClientConn {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Client")
	ret0, _ := ret[0].(*grpc.ClientConn)
	return ret0
}

// Client indicates an expected call of Client.
func (mr *MockConnMockRecorder) Client() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Client", reflect.TypeOf((*MockConn)(nil).Client))
}

// Close mocks base method.
func (m *MockConn) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockConnMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockConn)(nil).Close))
}

// validState mocks base method.
func (m *MockConn) validState() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "validState")
	ret0, _ := ret[0].(bool)
	return ret0
}

// validState indicates an expected call of validState.
func (mr *MockConnMockRecorder) validState() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "validState", reflect.TypeOf((*MockConn)(nil).validState))
}
