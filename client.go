// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

const maxRedirects = 50

// Client is an HTTP client for JSON based services. L is the type failure
// responses (non 2xx status codes) are decoded to. If L is string, the body
// of a failure response is used verbatim.
//
// A Client is safe for concurrent use and should be reused, as it pools
// connections.
type Client[L any] struct {
	httpClient         *http.Client
	logger             *slog.Logger
	allowUnknownFields bool
}

type config struct {
	retries              int
	withoutSSLValidation bool
	trustedCertificates  []string
	allowUnknownFields   bool
	logger               *slog.Logger
	transport            http.RoundTripper
}

// Option configures a [Client].
type Option func(*config)

// WithRetries sets how often a request is retried after a transport error.
// Negative values are treated as 0, which is the default.
func WithRetries(numberOfRetries int) Option {
	return func(c *config) { c.retries = numberOfRetries }
}

// WithoutSSLValidation disables the verification of server certificates and
// host names if set to true. It takes precedence over
// [WithTrustedCertificates].
func WithoutSSLValidation(withoutSSLValidation bool) Option {
	return func(c *config) { c.withoutSSLValidation = withoutSSLValidation }
}

// WithTrustedCertificates sets files with PEM or DER encoded certificates
// that are trusted in addition to the system's root certificates.
func WithTrustedCertificates(files ...string) Option {
	return func(c *config) { c.trustedCertificates = files }
}

// WithUnknownFieldsAllowed makes the client ignore JSON object fields in
// response bodies that have no counterpart in the type they are decoded to.
// By default such responses are rejected.
func WithUnknownFieldsAllowed() Option {
	return func(c *config) { c.allowUnknownFields = true }
}

// WithLogger sets the logger used for debug output. By default the logger
// returned by [slog.Default] is used.
func WithLogger(logger *slog.Logger) Option {
	return func(c *config) { c.logger = logger }
}

// withTransport replaces the underlying transport. It is used by tests.
func withTransport(transport http.RoundTripper) Option {
	return func(c *config) { c.transport = transport }
}

// New returns a client with failure type L.
func New[L any](opts ...Option) (*Client[L], error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	client := &Client[L]{logger: cfg.logger, allowUnknownFields: cfg.allowUnknownFields}
	if cfg.retries < 0 {
		client.log().Debug(fmt.Sprintf(
			"Number of retries must not be negative (was %d). Using default value \"0\".",
			cfg.retries))
		cfg.retries = 0
	}
	transport := cfg.transport
	if transport == nil {
		tlsCfg, err := tlsConfig(cfg.withoutSSLValidation, cfg.trustedCertificates)
		if err != nil {
			return nil, newError("Unable to configure certificate management", err)
		}
		t := http.DefaultTransport.(*http.Transport).Clone()
		if tlsCfg != nil {
			t.TLSClientConfig = tlsCfg
		}
		transport = t
	}
	if cfg.retries > 0 {
		transport = &retryTransport{base: transport, retries: cfg.retries}
	}
	client.httpClient = &http.Client{Transport: transport, CheckRedirect: checkRedirect}
	return client, nil
}

// NewWithRetries returns a client with failure type L and the given number
// of retries.
func NewWithRetries[L any](numberOfRetries int) (*Client[L], error) {
	return New[L](WithRetries(numberOfRetries))
}

// NewDefault returns a client with failure type string and no retries.
func NewDefault() (*Client[string], error) {
	return NewDefaultWithRetries(0)
}

// NewDefaultWithRetries returns a client with failure type string and the
// given number of retries.
func NewDefaultWithRetries(numberOfRetries int) (*Client[string], error) {
	return NewWithRetries[string](numberOfRetries)
}

// Close closes the idle connections held by the client.
func (c *Client[L]) Close() {
	c.httpClient.CloseIdleConnections()
}

// GetForObject sends a GET request and returns the response body decoded as
// R. A failure response is returned as an [*Error].
func GetForObject[R, L any](
	ctx context.Context, c *Client[L], url string, headers ...Header,
) (*R, error) {
	entity, err := GetForEntity[R](ctx, c, url, headers...)
	if err != nil {
		return nil, err
	}
	return entity.Get()
}

// GetForEntity sends a GET request and returns the response as a
// [ResponseEntity]. An error is returned only if no response could be
// obtained or decoded.
func GetForEntity[R, L any](
	ctx context.Context, c *Client[L], url string, headers ...Header,
) (ResponseEntity[L, R], error) {
	entity, err := exchange[R](ctx, c, http.MethodGet, url, nil, headers)
	if err != nil {
		return entity, newError("GET request failed.", err)
	}
	return entity, nil
}

// PostForObject sends a POST request with body encoded as JSON and returns
// the response body decoded as R. A failure response is returned as an
// [*Error].
func PostForObject[R, L any](
	ctx context.Context, c *Client[L], url string, body any, headers ...Header,
) (*R, error) {
	entity, err := PostForEntity[R](ctx, c, url, body, headers...)
	if err != nil {
		return nil, err
	}
	return entity.Get()
}

// PostForEntity sends a POST request with body encoded as JSON and returns
// the response as a [ResponseEntity]. An error is returned only if no
// response could be obtained or decoded.
func PostForEntity[R, L any](
	ctx context.Context, c *Client[L], url string, body any, headers ...Header,
) (ResponseEntity[L, R], error) {
	entity, err := exchange[R](ctx, c, http.MethodPost, url, &body, headers)
	if err != nil {
		return entity, newError("POST request failed.", err)
	}
	return entity, nil
}

// Put sends a PUT request with body encoded as JSON. A failure response is
// returned as an [*Error] whose cause describes the response.
func (c *Client[L]) Put(ctx context.Context, url string, body any, headers ...Header) error {
	if err := c.execute(ctx, http.MethodPut, url, &body, headers); err != nil {
		return newError("PUT request failed.", err)
	}
	return nil
}

// Delete sends a DELETE request. A failure response is returned as an
// [*Error] whose cause describes the response.
func (c *Client[L]) Delete(ctx context.Context, url string, headers ...Header) error {
	if err := c.execute(ctx, http.MethodDelete, url, nil, headers); err != nil {
		return newError("DELETE request failed.", err)
	}
	return nil
}

// execute sends a request whose success response carries no content of
// interest.
func (c *Client[L]) execute(
	ctx context.Context, method, url string, body *any, headers []Header,
) error {
	status, data, err := c.send(ctx, method, url, body, headers)
	if err != nil {
		return err
	}
	if isSuccess(status) {
		return nil
	}
	failure, err := decodeFailure[L](data, !c.allowUnknownFields)
	if err != nil {
		return err
	}
	_, err = Failed[L, struct{}](status, failure).Get()
	return err
}

func exchange[R, L any](
	ctx context.Context, c *Client[L], method, url string, body *any, headers []Header,
) (ResponseEntity[L, R], error) {
	status, data, err := c.send(ctx, method, url, body, headers)
	if err != nil {
		return ResponseEntity[L, R]{}, err
	}
	return handleResponse[L, R](status, data, !c.allowUnknownFields)
}

// send performs the request and returns the status code and the complete
// response body. A non-nil body is encoded as JSON.
func (c *Client[L]) send(
	ctx context.Context, method, url string, body *any, headers []Header,
) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(*body); err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(bytes.TrimSuffix(buf.Bytes(), []byte("\n")))
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
	}
	for _, h := range headers {
		req.Header.Add(h.Name, h.Value)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	if !isSuccess(resp.StatusCode) {
		c.log().DebugContext(ctx, fmt.Sprintf(
			"Request failed with status code <%d>: %s", resp.StatusCode, data))
	}
	return resp.StatusCode, data, nil
}

func (c *Client[L]) log() *slog.Logger {
	if c.logger != nil {
		return c.logger
	}
	return slog.Default()
}

// checkRedirect follows redirects of GET and HEAD requests, and "303 See
// Other" redirects of any request. Other redirect responses are returned to
// the caller as they are.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("maximum redirects (50) exceeded")
	}
	previous := via[len(via)-1].Method
	if previous == http.MethodGet || previous == http.MethodHead ||
		(req.Response != nil && req.Response.StatusCode == http.StatusSeeOther) {
		return nil
	}
	return http.ErrUseLastResponse
}
