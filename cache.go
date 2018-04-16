package doomsday

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

//Cache stores all the certificate data.
type Cache struct {
	store map[string]CacheObject
	lock  sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{store: map[string]CacheObject{}}
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

type PathList []string

//Multiple filters are "or"d together
type PathFilter struct {
	Under    []string
	Matching []string
}

func pathMatches(path, pattern string) bool {
	patternParts := strings.Split(pattern, "*")
	for i, p := range patternParts {
		patternParts[i] = regexp.QuoteMeta(p)
	}

	re := regexp.MustCompile(strings.Join(patternParts, `\A[^/:]*\Z`))
	return re.Match([]byte(path))
}

func pathIsUnder(path, dir string) bool {
	return strings.HasPrefix(strings.Trim(path, "/"), strings.TrimPrefix(dir, "/"))
}

//Doesn't modify reciever list
func (k PathList) Only(filter PathFilter) (ret PathList) {
OuterLoop:
	for _, key := range k {
		for _, match := range filter.Matching {
			if pathMatches(key, match) {
				ret = append(ret, key)
				continue OuterLoop
			}
		}

		for _, dir := range filter.Under {
			if pathIsUnder(key, dir) {
				ret = append(ret, key)
				continue OuterLoop
			}
		}
	}

	return
}

//Doesn't modify reciever list
func (k PathList) Except(filter PathFilter) (ret PathList) {
	for _, key := range k {
		var shouldNotAdd bool
		for _, match := range filter.Matching {
			if pathMatches(key, match) {
				shouldNotAdd = true
				goto DoneWithChecks
			}
		}

		for _, dir := range filter.Under {
			if pathIsUnder(key, dir) {
				shouldNotAdd = true
				goto DoneWithChecks
			}
		}

	DoneWithChecks:
		if !shouldNotAdd {
			ret = append(ret, key)
		}
	}

	return
}

type CacheObject struct {
	NotAfter time.Time
}
