package memory

import (
	"sync"
	"time"
)

type cacheStore struct {
	sync.RWMutex
	exit   chan bool
	values map[string]*cacheValue
}

type cacheValue struct {
	key     string
	value   interface{}
	expires time.Time
}

func (c *cacheStore) Get(key string) (interface{}, bool) {
	c.RLock()
	defer c.RUnlock()

	v, ok := c.values[key]
	if !ok {
		return nil, ok
	}
	if v.expires.IsZero() {
		return v, ok
	}
	if time.Now().After(v.expires) {
		return nil, false
	}
	return v.value, true
}

func (c *cacheStore) Set(key string, val interface{}, ttl time.Duration) {
	c.Lock()
	defer c.Unlock()

	c.values[key] = &cacheValue{
		key:     key,
		value:   val,
		expires: time.Now().Add(ttl),
	}
}

func (c *cacheStore) Delete(key string) {
	c.Lock()
	defer c.Unlock()
	delete(c.values, key)
}

func (c *cacheStore) List() map[string]*cacheValue {
	c.Lock()
	defer c.Unlock()
	return c.values
}

func (c *cacheStore) Close() {
	c.Lock()
	defer c.Unlock()

	select {
	case <-c.exit:
		return
	default:
		close(c.exit)
		c.values = map[string]*cacheValue{}
	}
}

func (c *cacheStore) run() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			c.Lock()

			for k, v := range c.values {
				if v.expires.IsZero() {
					continue
				}
				if time.Now().After(v.expires) {
					delete(c.values, k)
				}
			}

			c.Unlock()
		case <-c.exit:
			return
		}
	}
}

func newCache() *cacheStore {
	c := &cacheStore{
		exit:   make(chan bool),
		values: make(map[string]*cacheValue),
	}
	go c.run()
	return c
}
