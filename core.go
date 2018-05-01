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
	BasePath    string
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
	for _, path := range paths {
		secret, err := b.Backend.Get(path)
		if err != nil {
			return err
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

	b.SetCache(newCache)
	fmt.Println("Finished populating credentials")
	return nil
}

func (b *Core) Paths() (storage.PathList, error) {
	paths, err := b.Backend.List(b.BasePath)
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
