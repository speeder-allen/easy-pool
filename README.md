# easy-pool

a easy pool for golang

## test

[![Build Status](https://travis-ci.org/speeder-allen/easy-pool.svg?branch=master)](https://travis-ci.org/speeder-allen/easy-pool)

## install

```bash
go get -v github.com/speeder-allen/easy-pool
```

## how to use

this package define some pool, now version support:

- routines pool

- connection pool

### routines pool

a pool to limit routines number.

```golang
pool := easy_pool.NewRoutines(3) // 3 is capacity of the pool

var wg sync.WaitGroup

wg.Add(4)

for i := 0; i < 4; i ++ {
    go func(){
        defer wg.Done()
        pool.Take() // three routines will get pool, and one routines will block
    }()
}

go func(){
    time.Sleep(time.Second * 10)
    pool.Put(easy_pool.EmptyConn()) // after ten seconds ,a pool put to pool, so the last routines get pool and exit
}()

wg.Wait() // now will block until the last routine get pool and exit


```

