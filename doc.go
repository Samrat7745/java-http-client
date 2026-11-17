// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

// Package httpclient provides common functionality for the HTTP clients of the
// Carbyne Stack microservices.
//
// A [Client] sends JSON requests and maps each response either to a success
// value (2xx status codes) or to a failure value of the client's failure type
// L (any other status code). Both outcomes are carried by a [ResponseEntity].
//
//	client, err := httpclient.New[ErrorBody](
//		httpclient.WithRetries(3),
//		httpclient.WithTrustedCertificates("/etc/cs/ca.pem"),
//	)
//	if err != nil {
//		return err
//	}
//	secret, err := httpclient.GetForObject[Secret](ctx, client, "https://example.com/secret",
//		httpclient.BearerToken(token))
//
// Go methods cannot declare type parameters of their own, so the operations
// that return a typed response body are package-level functions that take the
// client as an argument. The response type has to be given explicitly; the
// failure type is inferred from the client.
package httpclient
