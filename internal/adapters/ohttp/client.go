package ohttp

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

const (
	DefaultMaxIdleConns          = 100
	DefaultMaxIdleConnsPerHost   = 10
	DefaultMaxConnsPerHost       = 100
	DefaultIdleConnTimeout       = 90 * time.Second
	DefaultDialTimeout           = 10 * time.Second
	DefaultKeepAlive             = 30 * time.Second
	DefaultTLSHandshakeTimeout   = 10 * time.Second
	DefaultResponseHeaderTimeout = 10 * time.Second
	DefaultExpectContinueTimeout = 1 * time.Second
	DefaultRequestTimeout        = 30 * time.Second
)

type Client struct {
	httpClient *http.Client
}

func DefaultTransport() *http.Transport {
	return createOptimizedTransport()
}

func NewClient(transport ...*http.Transport) *Client {
	var t *http.Transport
	if len(transport) > 0 && transport[0] != nil {
		t = transport[0]
	} else {
		t = DefaultTransport()
	}

	return &Client{
		httpClient: &http.Client{
			Transport: t,
			Timeout:   DefaultRequestTimeout,
		},
	}
}

func createOptimizedTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        DefaultMaxIdleConns,
		MaxIdleConnsPerHost: DefaultMaxIdleConnsPerHost,
		MaxConnsPerHost:     DefaultMaxConnsPerHost,
		IdleConnTimeout:     DefaultIdleConnTimeout,

		DialContext: (&net.Dialer{
			Timeout:   DefaultDialTimeout,
			KeepAlive: DefaultKeepAlive,
		}).DialContext,

		TLSHandshakeTimeout: DefaultTLSHandshakeTimeout,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: false,
		},
		ForceAttemptHTTP2:     true,
		ResponseHeaderTimeout: DefaultResponseHeaderTimeout,
		ExpectContinueTimeout: DefaultExpectContinueTimeout,

		DisableCompression: false,
		DisableKeepAlives:  false,
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}
