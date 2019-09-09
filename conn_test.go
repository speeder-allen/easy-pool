package easy_pool_test

import (
	"errors"
	easy_pool "github.com/speeder-allen/easy-pool"
	"gotest.tools/assert"
	"sync/atomic"
	"testing"
)

type mockconnection struct {
	closed uint32
}

var ErrorMockClose = errors.New("just a mock")

func (m *mockconnection) Close() error {
	atomic.CompareAndSwapUint32(&m.closed, 0, 1)
	return ErrorMockClose
}

type mockconn struct {
}

func TestConn_Close(t *testing.T) {
	mock := &mockconnection{closed: 0}
	c := easy_pool.Conn{Raw: mock}
	assert.Equal(t, mock.closed, uint32(0))
	err := c.Close()
	assert.Equal(t, err, ErrorMockClose)
	assert.Equal(t, mock.closed, uint32(1))
	m := mockconn{}
	cc := easy_pool.Conn{Raw: &m}
	err = cc.Close()
	assert.NilError(t, err)
}
