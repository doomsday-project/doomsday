package server

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"runtime"
	"sync"

	"github.com/thomasmmitchell/doomsday/storage"
)

type Core struct {
	Backend   storage.Accessor
	Name      string
	cache     *Cache
	cacheLock sync.RWMutex
}

type PopulateStats struct {
	NumPaths   int
	NumSuccess int
	NumCerts   int
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

func (b *Core) Populate() (*PopulateStats, error) {
	newCache := NewCache()
	paths, err := b.Backend.List()
	if err != nil {
		return nil, err
	}

	results, err := b.populateUsing(newCache, paths)
	if err != nil {
		return nil, err
	}

	b.SetCache(newCache)
	return results, nil
}

func (b *Core) populateUsing(cache *Cache, paths storage.PathList) (*PopulateStats, error) {
	if cache == nil {
		panic("Was given a nil cache")
	}

	var numWorkers = runtime.NumCPU() - 1
	if numWorkers < 1 {
		numWorkers = 1
	}
	if len(paths) < numWorkers {
		numWorkers = len(paths)
	}

	barrier := sync.WaitGroup{}
	barrier.Add(numWorkers)

	queue := make(chan string, len(paths))
	for _, path := range paths {
		queue <- path
	}
	close(queue)

	certCount := 0
	successCount := 0
	statLock := sync.Mutex{}

	fetch := func() {
		mySuccessCount, myCertCount := 0, 0
		for path := range queue {
			secret, err := b.Backend.Get(path)
			if err != nil {
				continue
			}

			for k, v := range secret {
				certs := parseCert(v)
				for _, cert := range certs {
					myCertCount++
					cache.Merge(
						fmt.Sprintf("%s", sha1.Sum(cert.Raw)),
						CacheObject{
							Subject:  cert.Subject,
							NotAfter: cert.NotAfter,
							Paths: []PathObject{
								{
									Location: path + ":" + k,
									Source:   b.Name,
								},
							},
						},
					)
				}
			}
			mySuccessCount++
		}
		statLock.Lock()
		successCount += mySuccessCount
		certCount += myCertCount
		statLock.Unlock()
		barrier.Done()
	}

	for i := 0; i < numWorkers; i++ {
		go fetch()
	}

	barrier.Wait()

	return &PopulateStats{
		NumPaths:   len(paths),
		NumSuccess: successCount,
		NumCerts:   certCount,
	}, nil
}

func parseCert(c string) []*x509.Certificate {
	certs := []*x509.Certificate{}
	//Populate a potential chain of certs (or even just one) into this here slice
	var pemBlock *pem.Block
	var rest = []byte(c)
	for {
		pemBlock, rest = pem.Decode(rest)
		//Skip over potential private keys in a cert chain.
		if pemBlock == nil {
			break
		}

		if pemBlock.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(pemBlock.Bytes)
		if err != nil {
			continue
		}

		certs = append(certs, cert)
		if len(rest) == 0 {
			break
		}
	}

	return certs
}
