// Code generated by MockGen. DO NOT EDIT.
// Source: ./mempool/mempool.go

// Package mock_mempool is a generated GoMock package.
package mock_mempool

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	types "github.com/tendermint/tendermint/abci/types"
	mempool "github.com/tendermint/tendermint/mempool"
	types0 "github.com/tendermint/tendermint/types"
)

// MockMempool is a mock of Mempool interface.
type MockMempool struct {
	ctrl     *gomock.Controller
	recorder *MockMempoolMockRecorder
}

// MockMempoolMockRecorder is the mock recorder for MockMempool.
type MockMempoolMockRecorder struct {
	mock *MockMempool
}

// NewMockMempool creates a new mock instance.
func NewMockMempool(ctrl *gomock.Controller) *MockMempool {
	mock := &MockMempool{ctrl: ctrl}
	mock.recorder = &MockMempoolMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMempool) EXPECT() *MockMempoolMockRecorder {
	return m.recorder
}

// CheckTx mocks base method.
func (m *MockMempool) CheckTx(tx types0.Tx, callback func(*types.Response), txInfo mempool.TxInfo) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckTx", tx, callback, txInfo)
	ret0, _ := ret[0].(error)
	return ret0
}

// CheckTx indicates an expected call of CheckTx.
func (mr *MockMempoolMockRecorder) CheckTx(tx, callback, txInfo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckTx", reflect.TypeOf((*MockMempool)(nil).CheckTx), tx, callback, txInfo)
}

// EnableTxsAvailable mocks base method.
func (m *MockMempool) EnableTxsAvailable() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "EnableTxsAvailable")
}

// EnableTxsAvailable indicates an expected call of EnableTxsAvailable.
func (mr *MockMempoolMockRecorder) EnableTxsAvailable() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnableTxsAvailable", reflect.TypeOf((*MockMempool)(nil).EnableTxsAvailable))
}

// Flush mocks base method.
func (m *MockMempool) Flush() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Flush")
}

// Flush indicates an expected call of Flush.
func (mr *MockMempoolMockRecorder) Flush() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Flush", reflect.TypeOf((*MockMempool)(nil).Flush))
}

// FlushAppConn mocks base method.
func (m *MockMempool) FlushAppConn() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FlushAppConn")
	ret0, _ := ret[0].(error)
	return ret0
}

// FlushAppConn indicates an expected call of FlushAppConn.
func (mr *MockMempoolMockRecorder) FlushAppConn() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FlushAppConn", reflect.TypeOf((*MockMempool)(nil).FlushAppConn))
}

// GetTxByKey mocks base method.
func (m *MockMempool) GetTxByKey(key types0.TxKey) (types0.Tx, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTxByKey", key)
	ret0, _ := ret[0].(types0.Tx)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// GetTxByKey indicates an expected call of GetTxByKey.
func (mr *MockMempoolMockRecorder) GetTxByKey(key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTxByKey", reflect.TypeOf((*MockMempool)(nil).GetTxByKey), key)
}

// Lock mocks base method.
func (m *MockMempool) Lock() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Lock")
}

// Lock indicates an expected call of Lock.
func (mr *MockMempoolMockRecorder) Lock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Lock", reflect.TypeOf((*MockMempool)(nil).Lock))
}

// ReapMaxBytesMaxGas mocks base method.
func (m *MockMempool) ReapMaxBytesMaxGas(maxBytes, maxGas int64) []types0.CachedTx {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReapMaxBytesMaxGas", maxBytes, maxGas)
	ret0, _ := ret[0].([]types0.CachedTx)
	return ret0
}

// ReapMaxBytesMaxGas indicates an expected call of ReapMaxBytesMaxGas.
func (mr *MockMempoolMockRecorder) ReapMaxBytesMaxGas(maxBytes, maxGas interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReapMaxBytesMaxGas", reflect.TypeOf((*MockMempool)(nil).ReapMaxBytesMaxGas), maxBytes, maxGas)
}

// ReapMaxTxs mocks base method.
func (m *MockMempool) ReapMaxTxs(max int) types0.Txs {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReapMaxTxs", max)
	ret0, _ := ret[0].(types0.Txs)
	return ret0
}

// ReapMaxTxs indicates an expected call of ReapMaxTxs.
func (mr *MockMempoolMockRecorder) ReapMaxTxs(max interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReapMaxTxs", reflect.TypeOf((*MockMempool)(nil).ReapMaxTxs), max)
}

// RemoveTxByKey mocks base method.
func (m *MockMempool) RemoveTxByKey(txKey types0.TxKey) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveTxByKey", txKey)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveTxByKey indicates an expected call of RemoveTxByKey.
func (mr *MockMempoolMockRecorder) RemoveTxByKey(txKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveTxByKey", reflect.TypeOf((*MockMempool)(nil).RemoveTxByKey), txKey)
}

// Size mocks base method.
func (m *MockMempool) Size() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Size")
	ret0, _ := ret[0].(int)
	return ret0
}

// Size indicates an expected call of Size.
func (mr *MockMempoolMockRecorder) Size() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Size", reflect.TypeOf((*MockMempool)(nil).Size))
}

// SizeBytes mocks base method.
func (m *MockMempool) SizeBytes() int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SizeBytes")
	ret0, _ := ret[0].(int64)
	return ret0
}

// SizeBytes indicates an expected call of SizeBytes.
func (mr *MockMempoolMockRecorder) SizeBytes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SizeBytes", reflect.TypeOf((*MockMempool)(nil).SizeBytes))
}

// TxsAvailable mocks base method.
func (m *MockMempool) TxsAvailable() <-chan struct{} {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TxsAvailable")
	ret0, _ := ret[0].(<-chan struct{})
	return ret0
}

// TxsAvailable indicates an expected call of TxsAvailable.
func (mr *MockMempoolMockRecorder) TxsAvailable() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TxsAvailable", reflect.TypeOf((*MockMempool)(nil).TxsAvailable))
}

// Unlock mocks base method.
func (m *MockMempool) Unlock() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Unlock")
}

// Unlock indicates an expected call of Unlock.
func (mr *MockMempoolMockRecorder) Unlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unlock", reflect.TypeOf((*MockMempool)(nil).Unlock))
}

// Update mocks base method.
func (m *MockMempool) Update(blockHeight int64, blockTxs types0.Txs, deliverTxResponses []*types.ResponseDeliverTx, newPreFn mempool.PreCheckFunc, newPostFn mempool.PostCheckFunc) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", blockHeight, blockTxs, deliverTxResponses, newPreFn, newPostFn)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockMempoolMockRecorder) Update(blockHeight, blockTxs, deliverTxResponses, newPreFn, newPostFn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockMempool)(nil).Update), blockHeight, blockTxs, deliverTxResponses, newPreFn, newPostFn)
}

// WasRecentlyEvicted mocks base method.
func (m *MockMempool) WasRecentlyEvicted(key types0.TxKey) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WasRecentlyEvicted", key)
	ret0, _ := ret[0].(bool)
	return ret0
}

// WasRecentlyEvicted indicates an expected call of WasRecentlyEvicted.
func (mr *MockMempoolMockRecorder) WasRecentlyEvicted(key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WasRecentlyEvicted", reflect.TypeOf((*MockMempool)(nil).WasRecentlyEvicted), key)
}
