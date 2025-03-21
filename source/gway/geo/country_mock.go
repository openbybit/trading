// Code generated by MockGen. DO NOT EDIT.
// Source: country.go

// Package geo is a generated GoMock package.
package geo

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCountry is a mock of Country interface.
type MockCountry struct {
	ctrl     *gomock.Controller
	recorder *MockCountryMockRecorder
}

// MockCountryMockRecorder is the mock recorder for MockCountry.
type MockCountryMockRecorder struct {
	mock *MockCountry
}

// NewMockCountry creates a new mock instance.
func NewMockCountry(ctrl *gomock.Controller) *MockCountry {
	mock := &MockCountry{ctrl: ctrl}
	mock.recorder = &MockCountryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCountry) EXPECT() *MockCountryMockRecorder {
	return m.recorder
}

// GetCurrencyCode mocks base method.
func (m *MockCountry) GetCurrencyCode() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCurrencyCode")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetCurrencyCode indicates an expected call of GetCurrencyCode.
func (mr *MockCountryMockRecorder) GetCurrencyCode() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCurrencyCode", reflect.TypeOf((*MockCountry)(nil).GetCurrencyCode))
}

// GetGeoNameID mocks base method.
func (m *MockCountry) GetGeoNameID() int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGeoNameID")
	ret0, _ := ret[0].(int64)
	return ret0
}

// GetGeoNameID indicates an expected call of GetGeoNameID.
func (mr *MockCountryMockRecorder) GetGeoNameID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGeoNameID", reflect.TypeOf((*MockCountry)(nil).GetGeoNameID))
}

// GetISO mocks base method.
func (m *MockCountry) GetISO() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetISO")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetISO indicates an expected call of GetISO.
func (mr *MockCountryMockRecorder) GetISO() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetISO", reflect.TypeOf((*MockCountry)(nil).GetISO))
}

// GetISO3 mocks base method.
func (m *MockCountry) GetISO3() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetISO3")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetISO3 indicates an expected call of GetISO3.
func (mr *MockCountryMockRecorder) GetISO3() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetISO3", reflect.TypeOf((*MockCountry)(nil).GetISO3))
}
