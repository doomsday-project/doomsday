package auth

import (
	"net/http"
)

type Authorizer interface {
	Configure(map[string]string) error
	LoginHandler() http.HandlerFunc
	TokenHandler() TokenFunc
	Identifier() AuthType
}

type TokenFunc func(http.HandlerFunc) http.HandlerFunc

type AuthType string
