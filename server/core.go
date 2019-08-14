package server

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/doomsday-project/doomsday/storage"
	yaml "gopkg.in/yaml.v2"
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

type x509CertWrapper struct {
	path string
	cert *x509.Certificate
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
				certs := wrapCerts(parseCert(v), k)
				if len(certs) == 0 {
					keys, err := parseYAMLKeys(v)
					if err == nil {
						for _, str := range keys {
							certs = append(certs, wrapCerts(parseCert(str.Value), k+":"+str.Path)...)
						}
					}
				}
				for _, cert := range certs {
					myCertCount++
					cache.Merge(
						fmt.Sprintf("%s", sha1.Sum(cert.cert.Raw)),
						CacheObject{
							Subject:  cert.cert.Subject,
							NotAfter: cert.cert.NotAfter,
							Paths: []PathObject{
								{
									Location: path + ":" + cert.path,
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

func wrapCerts(certs []*x509.Certificate, path string) (ret []x509CertWrapper) {
	for _, c := range certs {
		ret = append(ret, x509CertWrapper{
			path: path,
			cert: c,
		})
	}

	return ret
}

type YAMLKey struct {
	Path  string
	Value string
}

func parseYAMLKeys(y string) ([]YAMLKey, error) {
	var output interface{}
	err := yaml.Unmarshal([]byte(y), &output)
	if err != nil {
		return nil, err
	}

	ret := recurseTree(output, nil)

	return ret, nil
}

func recurseTree(obj interface{}, curPath []string) (ret []YAMLKey) {
	switch t := obj.(type) {
	case string:
		ret = append(ret, YAMLKey{
			Path:  treePath(curPath),
			Value: t,
		})
	case map[interface{}]interface{}:
		for k, v := range t {
			var kAsString string
			switch t2 := k.(type) {
			case string:
				kAsString = t2
			case int:
				kAsString = strconv.Itoa(t2)
			case bool:
				if t2 {
					kAsString = "true"
				} else {
					kAsString = "false"
				}
			}
			ret = append(ret, recurseTree(v, append(curPath, kAsString))...)
		}
	case []interface{}:
		for i, v := range t {
			ret = append(ret, recurseTree(v, append(curPath, strconv.Itoa(i)))...)
		}
	}

	return ret
}

func treePath(path []string) string {
	if len(path) == 0 {
		return "(root)"
	}

	return strings.Join(path, ".")
}
