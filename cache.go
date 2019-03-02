package doomsday

import (
	"crypto/x509/pkix"
	"sort"
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

func (c *Cache) Read(key string) (CacheObject, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	ret, found := c.store[key]
	return ret, found
}

func (c *Cache) Store(key string, value CacheObject) {
	c.lock.Lock()
	sort.Slice(value.Paths, func(i, j int) bool { return value.Paths[i].LessThan(value.Paths[j]) })
	c.store[key] = value
	c.lock.Unlock()
}

//ApplyDiff calculates a diff between o and n, and then atomically inserts
//things new to "n" and deletes things from "o" that are no longer in "n".
func (c *Cache) ApplyDiff(o, n *Cache) {
	keysToDelete, keysToAdd := calcDiff(o, n)

	c.lock.Lock()
	for key, paths := range keysToDelete {
		c.deletePaths(key, paths)
	}

	for key, cacheObj := range keysToAdd {
		c.addNewFrom(key, cacheObj)
	}
	c.lock.Unlock()
}

func calcDiff(o, n *Cache) (toDelete map[string][]PathObject, toAdd map[string]CacheObject) {
	toDelete = map[string][]PathObject{}
	toAdd = map[string]CacheObject{}
	o.lock.RLock()
	n.lock.RLock()
	for oldKey := range o.store {
		if newCacheObj, isInNew := n.store[oldKey]; !isInNew {
			toDelete[oldKey] = o.store[oldKey].Paths
		} else {
			thisToDelete, thisToAdd := pathListDiff(o.store[oldKey].Paths, newCacheObj.Paths)
			toDelete[oldKey] = thisToDelete
			objToAdd := newCacheObj
			objToAdd.Paths = thisToAdd
			toAdd[oldKey] = objToAdd
		}
	}

	for newKey, newObj := range n.store {
		if _, isInOld := o.store[newKey]; !isInOld {
			toAdd[newKey] = newObj
		}
	}

	o.lock.RUnlock()
	n.lock.RUnlock()

	return
}

//Takes two sorted slices of PathObject (sorted according to PathObject.LessThan)
// The output lists will be sorted in the same way.
func pathListDiff(o, n []PathObject) (toDelete, toAdd []PathObject) {
	//The path lists should be sorted for this to work
	var oIdx, nIdx int
	for !(oIdx == len(o) && nIdx == len(n)) {
		switch {
		case nIdx == len(n):
			toDelete = append(toDelete, o[oIdx])
			oIdx++
		case oIdx == len(o):
			toAdd = append(toAdd, n[nIdx])
			nIdx++
		case o[oIdx] == n[nIdx]:
			oIdx++
			nIdx++
		case o[oIdx].LessThan(n[nIdx]):
			toDelete = append(toDelete, o[oIdx])
			oIdx++
		default:
			toAdd = append(toAdd, n[nIdx])
			nIdx++
		}
	}
	return
}

//toDelete must be sorted according to PathObject.LessThan
func (c *Cache) deletePaths(key string, toDelete []PathObject) {
	obj, found := c.store[key]
	if !found {
		return
	}

	workingCopy := obj.Paths
	var wIdx, dIdx int
ForLoop:
	for dIdx != len(toDelete) {
		switch {
		case wIdx == len(workingCopy):
			break ForLoop
		case workingCopy[wIdx] == toDelete[dIdx]:
			before := workingCopy[:wIdx]
			var rest []PathObject
			if wIdx+1 < len(workingCopy) {
				rest = workingCopy[wIdx+1:]
			}
			workingCopy = append(before, rest...)
			//Don't increment the index because we lost an entry
		case workingCopy[wIdx].LessThan(toDelete[dIdx]):
			wIdx++
		default:
			dIdx++
		}
	}

	if len(workingCopy) == 0 {
		delete(c.store, key)
	}
	obj.Paths = workingCopy
	c.store[key] = obj
}

func (c *Cache) Merge(key string, obj CacheObject) {
	c.lock.Lock()
	c.addNewFrom(key, obj)
	c.lock.Unlock()
}

func (c *Cache) addNewFrom(key string, obj CacheObject) {
	existing, found := c.store[key]
	if !found {
		c.store[key] = obj
		return
	}

	existingLen := len(existing.Paths)
	var oIdx, eIdx int
	for oIdx != len(obj.Paths) {
		switch {
		case eIdx == existingLen:
			existing.Paths = append(existing.Paths, obj.Paths[oIdx])
			oIdx++
		case obj.Paths[oIdx] == existing.Paths[eIdx]:
			oIdx++
			eIdx++
		case obj.Paths[oIdx].LessThan(existing.Paths[eIdx]):
			eIdx++
		default:
			existing.Paths = append(existing.Paths, obj.Paths[oIdx])
			oIdx++
		}
	}

	sort.Slice(existing.Paths, func(i, j int) bool { return existing.Paths[i].LessThan(existing.Paths[j]) })
	c.store[key] = existing
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
	Paths                 []PathObject
	Subject               pkix.Name
	BasicConstraintsValid bool
	DNSNames              []string
	IPAddresses           []string
	NotAfter              time.Time
	NotBefore             time.Time
}

type PathObject struct {
	Location string
	Source   string
}

func (lhs PathObject) LessThan(rhs PathObject) bool {
	if lhs.Source == rhs.Source {
		return lhs.Location < rhs.Location
	}

	return lhs.Source < rhs.Source
}
