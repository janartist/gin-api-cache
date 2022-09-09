package store

import (
	"errors"
	"sync"
	"time"
)

func NewMemoryStore() *memoryStore {
	return &memoryStore{items: make(map[string]map[string]*item)}
}

type memoryStore struct {
	sync.RWMutex
	items map[string]map[string]*item
}

type item struct {
	value      ResponseCache
	expireTime time.Time
	isForever  bool
}

func (c *memoryStore) Set(key string, k string, val *ResponseCache, ttl time.Duration) error {
	isForever := false
	expireTime := time.Now().Add(ttl)
	if ttl < 0 {
		isForever = true
	}
	c.Lock()
	c.items[key] = map[string]*item{
		k: {
			value:      *val,
			expireTime: expireTime,
			isForever:  isForever,
		},
	}
	c.Unlock()
	return nil
}
func (c *memoryStore) Get(key string, k string, val *ResponseCache) (err error) {
	c.Lock()
	defer func() {
		c.Unlock()
	}()
	if i, ok := c.items[key][k]; ok {
		if !i.isExpired() {
			*val = i.value
			val.Expire = i.expireTime.Sub(time.Now())
			return
		}
		delete(c.items[key], k)
		if len(c.items[key]) == 0 {
			delete(c.items, key)
		}
	}
	err = errors.New("val is nil")
	return
}
func (c *memoryStore) Remove(key string) error {
	c.Lock()
	delete(c.items, key)
	c.Unlock()
	return nil
}

// isExpired 判断对象是否过期
func (i *item) isExpired() bool {
	if i.isForever {
		return false
	}
	return i.expireTime.Unix() <= time.Now().Unix()
}
