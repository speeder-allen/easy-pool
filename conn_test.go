package easy_pool_test

import (
	"fmt"
	easy_pool "github.com/speeder-allen/easy-pool"
	"gotest.tools/assert"
	"testing"
)

type testconn struct {
	Closed bool
}

func (t *testconn) Close() error {
	fmt.Println("closed.")
	t.Closed = true
	return nil
}

func TestConn_Close(t *testing.T) {
	tt := testconn{}
	c := easy_pool.Conn{&tt}
	assert.Equal(t, tt.Closed, false)
	c.Close()
	assert.Equal(t, tt.Closed, true)
}
