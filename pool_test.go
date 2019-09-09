package easy_pool_test

import (
	pool "github.com/speeder-allen/easy-pool"
	"gotest.tools/assert"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRoutinesPool(t *testing.T) {
	capacity := 3
	p := pool.NewRoutines(capacity)
	assert.Equal(t, p.Capacity(), capacity)
	assert.Equal(t, p.String(), "pool.routines")
	var wg sync.WaitGroup
	var wg_all sync.WaitGroup
	nm := capacity + 2
	wg.Add(capacity)
	wg_all.Add(nm)
	var count uint32 = 0
	for i := 0; i < nm; i++ {
		go func(idx int) {
			defer wg_all.Done()
			p.Take()
			log.Printf("[worker-%d]get lock now count = %d ==> %d", idx, atomic.LoadUint32(&count), atomic.AddUint32(&count, 1))
			if atomic.LoadUint32(&count) <= uint32(capacity) {
				wg.Done()
			}
		}(i)
	}
	wg.Wait()
	assert.Equal(t, count, uint32(capacity))
	p.Put(pool.EmptyConn())
	time.Sleep(time.Microsecond * 500)
	assert.Equal(t, count, uint32(capacity+1))
	assert.NilError(t, p.Close())
	time.Sleep(time.Microsecond * 500)
	assert.Equal(t, count, uint32(capacity+2))
	wg_all.Wait()
}

type MockFactory struct {
	Status uint32
}

func (m *MockFactory) Create() (pool.Conn, error) {
	log.Println("factory call!")
	return pool.Conn{Raw: &mockconnection{closed: 0}}, nil
}

func (m *MockFactory) Check(conn pool.Conn) bool {
	switch m.Status {
	case MockCheckFailClosed:
		log.Println("mock connect is alive!")
		return true
	case MockCheckFailOpened:
		log.Println("mock connect is dead!")
		return false
	}
	return true
}

const MockCheckFailClosed uint32 = 0
const MockCheckFailOpened uint32 = 1

func TestConnPool(t *testing.T) {
	factory := MockFactory{Status: MockCheckFailClosed}
	cp := pool.NewPoolConn(3, 0, &factory, &factory)
	var wg sync.WaitGroup
	assert.Equal(t, cp.String(), "pool.connection_pool")
	assert.Equal(t, cp.Capacity(), 3)
	closeC := make(chan struct{})
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			cc, _ := cp.Take()
			log.Println("get cc")
			go func() {
				<-closeC
				log.Println("put cc")
				cp.Put(cc)
			}()
		}()
	}
	wg.Wait()
	assert.Equal(t, factory.Status, MockCheckFailClosed)
	assert.Equal(t, cp.Len(), int64(5))
	closeC <- struct{}{}
	closeC <- struct{}{}
	closeC <- struct{}{}
	closeC <- struct{}{}
	time.Sleep(time.Second * 1)
	assert.Equal(t, cp.Len(), int64(4))
	closeC <- struct{}{}
	time.Sleep(time.Second * 1)
	assert.Equal(t, cp.Len(), int64(3))
	cc, _ := cp.Take()
	time.Sleep(time.Second * 1)
	log.Println("mock check to opened")
	cp.Put(cc)
	atomic.StoreUint32(&factory.Status, MockCheckFailOpened)
	time.Sleep(time.Second * 1)
	cp.Take()
	assert.Equal(t, cp.Len(), int64(3))
	for i := 0; i < 4; i++ {
		_, err := cp.Take()
		assert.NilError(t, err)
	}
	cmm, _ := cp.Take()
	_, err := cp.Take()
	assert.Equal(t, err, pool.ErrorTakeTimeout)
	log.Println("mock check to closed")
	atomic.StoreUint32(&factory.Status, MockCheckFailClosed)
	cc, err = cp.Take()
	assert.Equal(t, err, pool.ErrorTakeTimeout)
	go func() {
		time.Sleep(time.Second * 1)
		log.Println("timeout put cc back")
		cp.Put(cmm)
	}()
	_, err = cp.Take()
	assert.NilError(t, err)
	assert.Equal(t, cp.Len(), int64(6))
	log.Println("pull conn back...")
	close(closeC)
	log.Println("close pool")
	cp.Put(cmm)
	cp.Put(cmm)
	cp.Close()
	_, err = cp.Take()
	assert.Equal(t, err, pool.ErrorPoolClosed)
	err = cp.Put(cmm)
	assert.Equal(t, err, pool.ErrorPoolClosed)
}
