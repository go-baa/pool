package pool

import (
	"fmt"
	"sync"
)

// Pool common connection pool
type Pool struct {
	// New create connection function
	New func() interface{}
	// Ping check connection is ok
	Ping func(interface{}) bool
	// Close close connection
	Close func(interface{})
	store chan interface{}
	mu    sync.Mutex
}

// New create a pool with capacity
func New(initCap, maxCap int, newFunc func() interface{}) (*Pool, error) {
	if maxCap == 0 || initCap > maxCap {
		return nil, fmt.Errorf("invalid capacity settings")
	}
	p := new(Pool)
	p.store = make(chan interface{}, maxCap)
	if newFunc != nil {
		p.New = newFunc
	}
	for i := 0; i < initCap; i++ {
		v, err := p.create()
		if err != nil {
			return p, err
		}
		p.store <- v
	}
	return p, nil
}

// Len returns current connections in pool
func (p *Pool) Len() int {
	return len(p.store)
}

// Get returns a conn form store or create one
func (p *Pool) Get() (interface{}, error) {
	if p.store == nil {
		// pool aleardy destroyed, returns new connection
		return p.create()
	}
	for {
		select {
		case v := <-p.store:
			if p.Ping != nil && p.Ping(v) == false {
				continue
			}
			return v, nil
		default:
			return p.create()
		}
	}
}

// Put set back conn into store again
func (p *Pool) Put(v interface{}) {
	select {
	case p.store <- v:
		return
	default:
		// pool is full, close passed connection
		if p.Close != nil {
			p.Close(v)
		}
		return
	}
}

// Destroy clear all connections
func (p *Pool) Destroy() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.store == nil {
		// pool aleardy destroyed
		return
	}
	close(p.store)
	for v := range p.store {
		if p.Close != nil {
			p.Close(v)
		}
	}
	p.store = nil
}

func (p *Pool) create() (interface{}, error) {
	if p.New == nil {
		return nil, fmt.Errorf("Pool.New is nil, can not create connection")
	}
	return p.New(), nil
}
