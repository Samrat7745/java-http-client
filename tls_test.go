// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

package httpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const (
	certificatePath = "testdata/cli_test.pem"
	keyPath         = "testdata/cli_test.key"
	testEndpoint    = "/test"
	successResponse = "success"
)

// newTLSServer starts an HTTPS server that identifies itself with the
// self-signed test certificate and returns the URL of its test endpoint.
func newTLSServer(t *testing.T) string {
	t.Helper()
	cert, err := tls.LoadX509KeyPair(certificatePath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc(testEndpoint, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"` + successResponse + `"`))
	})
	srv := httptest.NewUnstartedServer(mux)
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv.URL + testEndpoint
}

func TestSSL_TrustedCertificate_Succeeds(t *testing.T) {
	url := newTLSServer(t)
	client, err := New[string](WithTrustedCertificates(certificatePath))
	if err != nil {
		t.Fatal(err)
	}
	got, err := GetForObject[string](context.Background(), client, url)
	if err != nil || got == nil || *got != successResponse {
		t.Errorf("got %v, %v; want %q", got, err, successResponse)
	}
}

func TestSSL_ValidationDisabled_Succeeds(t *testing.T) {
	url := newTLSServer(t)
	client, err := New[string](WithoutSSLValidation(true))
	if err != nil {
		t.Fatal(err)
	}
	got, err := GetForObject[string](context.Background(), client, url)
	if err != nil || got == nil || *got != successResponse {
		t.Errorf("got %v, %v; want %q", got, err, successResponse)
	}
}

func TestSSL_NonExistingCertificate_FailsToBuildClient(t *testing.T) {
	_, err := New[string](WithTrustedCertificates(filepath.Join(t.TempDir(), "missing.pem")))
	var csErr *Error
	if !errors.As(err, &csErr) {
		t.Fatalf("expected *Error, got %T (%v)", err, err)
	}
	if csErr.Message() != "Unable to configure certificate management" {
		t.Errorf("message = %q", csErr.Message())
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("cause = %v, want fs.ErrNotExist", csErr.Unwrap())
	}
}

func TestSSL_InvalidCertificate_FailsToBuildClient(t *testing.T) {
	for name, content := range map[string]string{
		"garbage":      "this is not a certificate",
		"empty":        "",
		"key only":     "-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n",
		"broken block": "-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n",
	} {
		t.Run(name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "cert.pem")
			if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := New[string](WithTrustedCertificates(file))
			var csErr *Error
			if !errors.As(err, &csErr) {
				t.Errorf("expected *Error, got %T (%v)", err, err)
			}
		})
	}
}

func TestSSL_UntrustedCertificate_Fails(t *testing.T) {
	url := newTLSServer(t)
	client, err := NewDefaultWithRetries(2)
	if err != nil {
		t.Fatal(err)
	}
	_, err = GetForObject[string](context.Background(), client, url)
	var csErr *Error
	if !errors.As(err, &csErr) {
		t.Fatalf("expected *Error, got %T (%v)", err, err)
	}
	var unknownAuthority x509.UnknownAuthorityError
	if !errors.As(err, &unknownAuthority) {
		t.Errorf("cause = %v, want x509.UnknownAuthorityError", csErr.Unwrap())
	}
}

func TestSSL_ValidationDisabledTakesPrecedenceOverCertificates(t *testing.T) {
	cfg, err := tlsConfig(true, []string{"does-not-exist.pem"})
	if err != nil || cfg == nil || !cfg.InsecureSkipVerify {
		t.Errorf("tlsConfig() = %+v, %v", cfg, err)
	}
	if cfg, err := tlsConfig(false, nil); cfg != nil || err != nil {
		t.Errorf("tlsConfig() = %+v, %v; want defaults", cfg, err)
	}
}

func loadTestCertificate(t *testing.T) *x509.Certificate {
	t.Helper()
	data, err := os.ReadFile(certificatePath)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

// The pool has to trust what either the system or the given certificates
// trust. The system half is covered by the integration tests.
func TestCertPool_TrustsGivenCertificates(t *testing.T) {
	cert := loadTestCertificate(t)
	if system, err := x509.SystemCertPool(); err == nil {
		if _, err := cert.Verify(x509.VerifyOptions{Roots: system, DNSName: "localhost"}); err == nil {
			t.Fatal("test certificate is unexpectedly trusted by the system")
		}
	}
	derPath := filepath.Join(t.TempDir(), "cert.der")
	if err := os.WriteFile(derPath, cert.Raw, 0o600); err != nil {
		t.Fatal(err)
	}
	for name, file := range map[string]string{"PEM": certificatePath, "DER": derPath} {
		t.Run(name, func(t *testing.T) {
			pool, err := newCertPool([]string{file})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := cert.Verify(x509.VerifyOptions{Roots: pool, DNSName: "localhost"}); err != nil {
				t.Errorf("certificate not trusted: %v", err)
			}
		})
	}
}

func TestCertPool_ReadsAllCertificatesOfABundle(t *testing.T) {
	data, err := os.ReadFile(certificatePath)
	if err != nil {
		t.Fatal(err)
	}
	certs, err := parseCertificates(append(append([]byte{}, data...), data...))
	if err != nil || len(certs) != 2 {
		t.Errorf("parseCertificates() = %d certificates, %v; want 2", len(certs), err)
	}
}
