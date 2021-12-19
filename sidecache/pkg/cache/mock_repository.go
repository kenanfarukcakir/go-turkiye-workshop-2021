// Code generated by MockGen. DO NOT EDIT.
// Source: repository.go

// Package mock_cache is a generated GoMock package.
package cache

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockCacheRepository is a mock of CacheRepository interface.
type MockCacheRepository struct {
	ctrl     *gomock.Controller
	recorder *MockCacheRepositoryMockRecorder
}

// MockCacheRepositoryMockRecorder is the mock recorder for MockCacheRepository.
type MockCacheRepositoryMockRecorder struct {
	mock *MockCacheRepository
}

// NewMockCacheRepository creates a new mock instance.
func NewMockCacheRepository(ctrl *gomock.Controller) *MockCacheRepository {
	mock := &MockCacheRepository{ctrl: ctrl}
	mock.recorder = &MockCacheRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCacheRepository) EXPECT() *MockCacheRepositoryMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockCacheRepository) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockCacheRepositoryMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockCacheRepository)(nil).Close))
}

// Get mocks base method.
func (m *MockCacheRepository) Get(key string) []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", key)
	ret0, _ := ret[0].([]byte)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockCacheRepositoryMockRecorder) Get(key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCacheRepository)(nil).Get), key)
}

// Remove mocks base method.
func (m *MockCacheRepository) Remove(key string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Remove", key)
	ret0, _ := ret[0].(error)
	return ret0
}

// Remove indicates an expected call of Remove.
func (mr *MockCacheRepositoryMockRecorder) Remove(key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Remove", reflect.TypeOf((*MockCacheRepository)(nil).Remove), key)
}

// SetKey mocks base method.
func (m *MockCacheRepository) SetKey(key string, value []byte, ttl int) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetKey", key, value, ttl)
}

// SetKey indicates an expected call of SetKey.
func (mr *MockCacheRepositoryMockRecorder) SetKey(key, value, ttl interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetKey", reflect.TypeOf((*MockCacheRepository)(nil).SetKey), key, value, ttl)
}
