package easy_pool

import "io"

type Conn struct {
	Raw interface{}
}

func (c *Conn) Close() error {
	if c.Raw != nil {
		if r, ok := c.Raw.(io.Closer); ok {
			return r.Close()
		}

	}
	return nil
}
