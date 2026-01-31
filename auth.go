package main

import (
	"crypto/subtle"
	"net/http"
)

const apiKeyHeader = "X-Api-Key"

type AuthenticatedKey struct {
	ApiKey ApiKey
}

func authenticate(options Options, r *http.Request) (*AuthenticatedKey, bool) {
	provided := r.Header.Get(apiKeyHeader)
	if provided == "" {
		return nil, false
	}

	for _, key := range options.ApiKeys {
		if subtle.ConstantTimeCompare([]byte(key.Key), []byte(provided)) == 1 {
			return &AuthenticatedKey{ApiKey: key}, true
		}
	}

	return nil, false
}
