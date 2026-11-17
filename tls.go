// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

package httpclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// tlsConfig returns the TLS configuration for the given options, or nil if
// the defaults apply.
func tlsConfig(withoutSSLValidation bool, trustedCertificates []string) (*tls.Config, error) {
	switch {
	case withoutSSLValidation:
		return &tls.Config{InsecureSkipVerify: true}, nil //nolint:gosec // explicitly requested
	case len(trustedCertificates) > 0:
		pool, err := newCertPool(trustedCertificates)
		if err != nil {
			return nil, err
		}
		return &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}, nil
	default:
		return nil, nil
	}
}

// newCertPool returns a pool holding the system's root certificates together
// with the certificates read from the given files. A server is therefore
// trusted if either the system or one of the given certificates vouches for it.
func newCertPool(files []string) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		certs, err := parseCertificates(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		for _, cert := range certs {
			pool.AddCert(cert)
		}
	}
	return pool, nil
}

// parseCertificates reads all certificates from PEM or DER encoded data.
func parseCertificates(data []byte) ([]*x509.Certificate, error) {
	if !bytes.Contains(data, []byte("-----BEGIN")) {
		certs, err := x509.ParseCertificates(data)
		if err == nil && len(certs) == 0 {
			err = fmt.Errorf("no certificate found")
		}
		return certs, err
	}
	var certs []*x509.Certificate
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificate found")
	}
	return certs, nil
}
