package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
)

type Backend interface {
	List(path string) ([]string, error)
	Get(path string) (map[string]string, error)
}

type BackendCore struct {
	Backend  Backend
	Cache    *Cache
	BasePath string
}

func (b *BackendCore) Populate() error {
	paths, err := b.Paths()
	if err != nil {
		return err
	}
	return b.PopulateUsing(paths)
}

func (b *BackendCore) PopulateUsing(paths PathList) error {
	for _, path := range paths {
		secret, err := b.Backend.Get(path)
		if err != nil {
			return err
		}
		for k, v := range secret {
			cert := parseCert(v)
			if cert != nil {
				b.Cache.Store(fmt.Sprintf("%s:%s", path, k), CacheObject{NotAfter: cert.NotAfter})
			}
		}
	}
	return nil
}

func (b *BackendCore) Paths() (PathList, error) {
	paths, err := b.recursivelyList(b.BasePath)
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (b *BackendCore) recursivelyList(path string) (PathList, error) {
	var leaves []string
	list, err := b.Backend.List(path)
	if err != nil {
		return nil, err
	}

	for _, v := range list {
		if !strings.HasSuffix(v, "/") {
			leaves = append(leaves, canonizePath(fmt.Sprintf("%s/%s", path, v)))
		} else {
			rList, err := b.recursivelyList(canonizePath(fmt.Sprintf("%s/%s", path, v)))
			if err != nil {
				return nil, err
			}
			leaves = append(leaves, rList...)
		}
	}

	return leaves, nil
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

func canonizePath(path string) string {
	pathChunks := strings.Split(path, "/")
	for i := 0; i < len(pathChunks); i++ {
		if pathChunks[i] == "" {
			pathChunks = append(pathChunks[:i], pathChunks[i+1:]...)
			i--
		}
	}
	return strings.Join(pathChunks, "/")
}
