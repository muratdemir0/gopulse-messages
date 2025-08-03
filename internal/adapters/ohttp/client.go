package ohttp

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	HeaderRetryAttempt = "X-Retry-Attempt"
)

const (
	DefaultMaxRetries          = 5
	DefaultInitialInterval     = 100 * time.Millisecond
	DefaultRandomizationFactor = 0.5
	DefaultMultiplier          = 1.5
	DefaultMaxInterval         = 10 * time.Second
	DefaultMaxElapsedTime      = 15 * time.Second
)

type RetryConfig struct {
	MaxRetries          uint64
	InitialInterval     time.Duration
	RandomizationFactor float64
	Multiplier          float64
	MaxInterval         time.Duration
	MaxElapsedTime      time.Duration
}

type Config struct {
	Transport           *http.Transport
	RetryConfig         *RetryConfig
	EnableOpenTelemetry bool
}

type Client struct {
	httpClient  *http.Client
	retryConfig *RetryConfig
}

func DefaultTransport() *http.Transport {
	return createOptimizedTransport()
}

func NewClient(configs ...Config) *Client {
	cfg := Config{
		Transport:           DefaultTransport(),
		RetryConfig:         nil,
		EnableOpenTelemetry: true,
	}

	if len(configs) > 0 {
		userConfig := configs[0]
		if userConfig.Transport != nil {
			cfg.Transport = userConfig.Transport
		}
		if userConfig.RetryConfig != nil {
			cfg.RetryConfig = userConfig.RetryConfig
		}
		cfg.EnableOpenTelemetry = userConfig.EnableOpenTelemetry
	}

	var transport http.RoundTripper = cfg.Transport
	if cfg.EnableOpenTelemetry {
		transport = otelhttp.NewTransport(cfg.Transport)
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   DefaultRequestTimeout,
		},
		retryConfig: cfg.RetryConfig,
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
	if c.retryConfig == nil {
		return c.httpClient.Do(req)
	}
	return c.doWithRetry(req)
}

func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var attempt int

	operation := func() error {
		attempt++
		var err error
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %s", resp.Status)
		}

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return backoff.Permanent(fmt.Errorf("client error: %s", resp.Status))
		}

		return nil
	}

	bo := backoff.NewExponentialBackOff()
	if c.retryConfig.InitialInterval > 0 {
		bo.InitialInterval = c.retryConfig.InitialInterval
	}
	if c.retryConfig.RandomizationFactor > 0 {
		bo.RandomizationFactor = c.retryConfig.RandomizationFactor
	}
	if c.retryConfig.Multiplier > 0 {
		bo.Multiplier = c.retryConfig.Multiplier
	}
	if c.retryConfig.MaxInterval > 0 {
		bo.MaxInterval = c.retryConfig.MaxInterval
	}
	if c.retryConfig.MaxElapsedTime > 0 {
		bo.MaxElapsedTime = c.retryConfig.MaxElapsedTime
	}

	b := backoff.WithMaxRetries(bo, c.retryConfig.MaxRetries)
	err := backoff.Retry(operation, backoff.WithContext(b, req.Context()))

	if err != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", attempt, err)
	}

	if resp != nil {
		resp.Header.Set(HeaderRetryAttempt, strconv.Itoa(attempt))
	}

	return resp, nil
}
