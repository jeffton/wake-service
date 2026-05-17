package main

import (
	"crypto/subtle"
	"net/http"
)

const apiKeyHeader = "X-Api-Key"

const (
	apiKeyTypeWeather = "weather"
	apiKeyTypeFull    = "full"
)

type AuthenticatedKey struct {
	ApiKey ApiKey
}

func (key *AuthenticatedKey) IsFull() bool {
	return key.ApiKey.Type == apiKeyTypeFull
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
