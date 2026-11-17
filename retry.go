// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

package httpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
)

// retryTransport re-sends a request that failed with a transport error, up to
// retries additional times.
type retryTransport struct {
	base    http.RoundTripper
	retries int
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		resp, err := t.base.RoundTrip(req)
		if err == nil || attempt >= t.retries || !retriable(req, err) {
			return resp, err
		}
	}
}

// retriable reports whether a failed request may be sent again. Only requests
// without a body are retried, and only for errors that are neither timeouts
// or cancellations, nor name resolution, connection establishment or TLS
// failures.
func retriable(req *http.Request, err error) bool {
	if req.Body != nil && req.Body != http.NoBody {
		return false
	}
	if req.Context().Err() != nil ||
		errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return false
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && (opErr.Op == "dial" || opErr.Op == "remote error") {
		return false
	}
	return !isTLSError(err)
}

func isTLSError(err error) bool {
	var (
		verification *tls.CertificateVerificationError
		recordHeader tls.RecordHeaderError
		alert        tls.AlertError
		unknownAuth  x509.UnknownAuthorityError
		hostname     x509.HostnameError
		invalid      x509.CertificateInvalidError
	)
	return errors.As(err, &verification) || errors.As(err, &recordHeader) ||
		errors.As(err, &alert) || errors.As(err, &unknownAuth) ||
		errors.As(err, &hostname) || errors.As(err, &invalid)
}
