package easy_pool

import (
	"io"
	"time"
)

type Conn struct {
	UUID string
	Raw  interface{}
	nsec uint32
	sec  int64
}

func (c *Conn) Close() error {
	if c.Raw != nil {
		if r, ok := c.Raw.(io.Closer); ok {
			return r.Close()
		}
	}
	return nil
}

func (c *Conn) Time() time.Time {
	return time.Unix(c.sec, int64(c.nsec))
}
