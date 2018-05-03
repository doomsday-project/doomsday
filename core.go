package doomsday

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"

	"github.com/thomasmmitchell/doomsday/storage"
)

type Core struct {
	Backend     storage.Accessor
	cache       *Cache
	cacheLock   sync.RWMutex
	BackendName string
}

func (b *Core) SetCache(c *Cache) {
	b.cacheLock.Lock()
	defer b.cacheLock.Unlock()
	c.lock = &b.cacheLock
	b.cache = c
}

func (b *Core) Cache() *Cache {
	return b.cache
}

func (b *Core) Populate() error {
	paths, err := b.Paths()
	if err != nil {
		return err
	}
	return b.PopulateUsing(paths)
}

func (b *Core) PopulateUsing(paths storage.PathList) error {
	fmt.Println("Began populating credentials")
	newCache := NewCache()

	const numWorkers = 4

	queue := make(chan string, len(paths))
	for _, path := range paths {
		queue <- path
	}
	close(queue)

	doneChan := make(chan bool, numWorkers)
	errChan := make(chan error, numWorkers)

	fetch := func() {
		for path := range queue {
			secret, err := b.Backend.Get(path)
			if err != nil {
				errChan <- err
			}

			for _, v := range secret {
				cert := parseCert(v)
				if cert != nil {
					newCache.Store(path,
						CacheObject{
							Subject:  cert.Subject,
							NotAfter: cert.NotAfter,
						},
					)
					//Don't get multiple certs from within the same secret - they're probably
					// the same one
					break
				}
			}
		}
		doneChan <- true
	}

	for i := 0; i < numWorkers; i++ {
		go fetch()
	}

	var err error
	numDone := 0
	for {
		select {
		case <-doneChan:
			numDone++
			if numDone == numWorkers {
				goto doneWaiting
			}
		case err = <-errChan:
			goto doneWaiting
		}
	}
doneWaiting:
	if err != nil {
		return err
	}

	b.SetCache(newCache)
	fmt.Println("Finished populating credentials")
	return nil
}

func (b *Core) Paths() (storage.PathList, error) {
	paths, err := b.Backend.List()
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func parseCert(c string) *x509.Certificate {
	pemBlock, _ := pem.Decode([]byte(c))
	if pemBlock == nil {
		return nil
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil
	}

	return cert
}
