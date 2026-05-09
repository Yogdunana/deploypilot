package util

import (
	"net/http"
	"time"
)

// DefaultClient is an HTTP client with sensible timeouts.
var DefaultClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}
