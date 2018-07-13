package doomsday

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"

	"github.com/thomasmmitchell/doomsday/storage"
)

type Core struct {
	Backends    []storage.Accessor
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
	newCache := NewCache()
	for _, accessor := range b.Backends {
		fmt.Printf("Enumerating possible paths for accessor `%s'\n", accessor.Name())
		paths, err := accessor.List()
		if err != nil {
			return err
		}

		fmt.Printf("Found %d paths to look up\n", len(paths))
		err = b.populateUsing(accessor, newCache, paths)
		if err != nil {
			return err
		}
	}

	b.SetCache(newCache)
	return nil
}

func (b *Core) populateUsing(backend storage.Accessor, cache *Cache, paths storage.PathList) error {
	fmt.Println("Began populating credentials")
	if cache == nil {
		panic("Was given a nil cache")
	}

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
			secret, err := backend.Get(path)
			if err != nil {
				errChan <- err
			}

			for _, v := range secret {
				cert := parseCert(v)
				if cert != nil {
					cache.Store(path,
						CacheObject{
							Backend:  backend.Name(),
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

	fmt.Println("Finished populating credentials")
	return nil
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
