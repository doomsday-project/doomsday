package doomsday

import (
	"crypto/x509/pkix"
	"sync"
	"time"
)

//Cache stores all the certificate data.
type Cache struct {
	store map[string]CacheObject
	lock  *sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		store: map[string]CacheObject{},
		lock:  &sync.RWMutex{}, //Lock for writing while the cache is being populated
	}
}

//Keys returns a list of all of the keys in the cache
func (c *Cache) Keys() []string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	ret := make([]string, 0, len(c.store))
	for key := range c.store {
		ret = append(ret, key)
	}
	return ret
}

func (c *Cache) Read(path string) (CacheObject, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	ret, found := c.store[path]
	return ret, found
}

func (c *Cache) Store(path string, value CacheObject) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.store[path] = value
}

func (c *Cache) Map() map[string]CacheObject {
	c.lock.RLock()
	defer c.lock.RUnlock()
	ret := make(map[string]CacheObject, len(c.store))
	for k, v := range c.store {
		ret[k] = v
	}
	return ret
}

type CacheObject struct {
	Subject  pkix.Name
	NotAfter time.Time
}
