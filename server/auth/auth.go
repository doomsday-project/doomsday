package auth

import (
	"net/http"
)

type Authorizer interface {
	Configure(map[string]string) error
	LoginHandler() http.HandlerFunc
	TokenHandler() TokenFunc
}

type TokenFunc func(http.HandlerFunc) http.HandlerFunc

func nopAuth(fn http.HandlerFunc) http.HandlerFunc { return fn }
