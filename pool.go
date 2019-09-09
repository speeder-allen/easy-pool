package easy_pool

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type Pool interface {
	Take() (Conn, error)
	Put(conn Conn) error
	Capacity() int
	io.Closer
	fmt.Stringer
}

type poolRoutines struct {
	ch   chan struct{}
	cap  int
	once sync.Once
}

func (r *poolRoutines) Take() (Conn, error) {
	<-r.ch
	return emptyConn, nil
}

func (r *poolRoutines) Put(conn Conn) error {
	r.ch <- struct{}{}
	return nil
}

func (r *poolRoutines) Capacity() int {
	return r.cap
}

func (r *poolRoutines) Close() error {
	r.once.Do(func() {
		close(r.ch)
	})
	return nil
}

func (r *poolRoutines) String() string {
	return "pool.routines"
}

// NewRoutines is create a goroutines pool
// it's use to limit goroutines
func NewRoutines(capacity int) *poolRoutines {
	c := poolRoutines{cap: capacity, ch: make(chan struct{}, capacity)}
	for i := 0; i < c.cap; i++ {
		c.ch <- struct{}{}
	}
	return &c
}

type ConnectionCreator interface {
	Create() (Conn, error)
}

type ConnectionChecker interface {
	Check(conn Conn) bool
}

var ErrorTakeTimeout = errors.New("get connection from pool timeout")

var ErrorPoolClosed = errors.New("pool is closed")

var ErrorConNil = errors.New("connection is nil")

var emptyConn = Conn{}

type poolConn struct {
	ch      chan Conn
	cap     int
	factory ConnectionCreator
	checker ConnectionChecker

	max    int64
	len    int64
	closed int32
}

// NewPoolConn will create a connection pool
// capacity is pool capacity
// limit is max connection, it must bigger or equal capacity , default limit = 2 x capacity
//    if pool take empty and not put yet,
//	  factory will build new connection (not bigger then limit),
//    when connection put to pool, if pool full ,the connection will close.
func NewPoolConn(capacity int, limit int, factory ConnectionCreator, checker ConnectionChecker) *poolConn {
	if limit < capacity {
		limit = capacity * 2
	}
	p := poolConn{cap: capacity, factory: factory, checker: checker, ch: make(chan Conn, capacity), max: int64(limit), len: 0, closed: 0}
	return &p
}

func (r *poolConn) Len() int64 {
	return atomic.LoadInt64(&r.len)
}

func (p *poolConn) Take() (Conn, error) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return emptyConn, ErrorPoolClosed
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
loop:
	for {
		select {
		case conn := <-p.ch:
			if !p.checker.Check(conn) {
				return createConnWithFactory(p.factory)
			}
			return conn, nil
		case <-ctx.Done():
			break loop
		default:
			length := atomic.LoadInt64(&p.len)
			if length < p.max {
				atomic.AddInt64(&p.len, 1)
				return createConnWithFactory(p.factory)
			}
			time.Sleep(time.Microsecond * 10)
		}
	}
	return emptyConn, ErrorTakeTimeout
}

func (p *poolConn) Put(conn Conn) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		defer conn.Close()
		return ErrorPoolClosed
	}
	if conn == emptyConn {
		return ErrorConNil
	}
	select {
	case p.ch <- conn:
		return nil
	default:
		atomic.AddInt64(&p.len, -1)
		return conn.Close()
	}
}

func (p *poolConn) Capacity() int {
	return p.cap
}

func (p *poolConn) Close() error {
	if atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		var err error
		for i := 0; i < p.cap; i++ {
			select {
			case conn := <-p.ch:
				err = conn.Close()
			default:
			}
		}
		close(p.ch)
		return err
	}
	return nil
}

func (r *poolConn) String() string {
	return "pool.connection_pool"
}

func createConnWithFactory(factory ConnectionCreator) (Conn, error) {
	conn, err := factory.Create()
	if err != nil {
		return emptyConn, err
	}
	n := time.Now()
	conn.sec = n.Unix()
	conn.nsec = uint32(n.Nanosecond())
	conn.UUID = uuid.New().String()
	return conn, nil
}

func EmptyConn() Conn {
	return emptyConn
}
