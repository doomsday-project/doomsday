package doomsday

import (
	"crypto/x509"
	"encoding/pem"
	"runtime"
	"sync"

	"github.com/thomasmmitchell/doomsday/storage"
)

type Core struct {
	Backend   storage.Accessor
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

			for _, v := range secret {
				cert := parseCert(v)
				if cert != nil {
					myCertCount++
					cache.Store(path,
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

func parseCert(c string) *x509.Certificate {
	certChain := []*x509.Certificate{}
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

		certChain = append(certChain, cert)
		if len(rest) == 0 {
			break
		}
	}

	return getLeafCert(certChain)
}

//I'm assuming that given a cert chain, the chain is either from server to
//highest intermediate/root CA, or in the reverse order. You know - not just
//some random smattering of certs, because that doesn't work with anything I
//know of. Therefore, I check the first two certs in the chain
//for signature direction to try to determine which is the server cert, and
//then return only that. If the thing that signed it is in the chain and we
//can actually control it, its probably somewhere else in the storage and
//it'll be caught at another key.
func getLeafCert(chain []*x509.Certificate) *x509.Certificate {
	if len(chain) == 0 {
		return nil
	}

	//This applies if the chain is len 1, or if the chain is leaf to root
	ret := chain[0]

	if len(chain) > 1 {
		//Check if the first cert signed the second cert
		rootToLeaf := chain[0].CheckSignature(
			chain[1].SignatureAlgorithm,
			chain[1].RawTBSCertificate,
			chain[1].Signature) == nil

		if rootToLeaf {
			ret = chain[len(chain)-1]
		}
	}

	return ret
}
