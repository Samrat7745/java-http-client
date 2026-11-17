// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package httpclient

import (
	"context"
	"net/http"
	"testing"
)

// These tests need access to the internet. Run them with
//
//	go test -tags integration ./...

const googleDirectionsRestURI = "https://maps.googleapis.com/maps/api/directions/json"

type googleDirectionsResponse struct {
	ErrorMessage string `json:"error_message"`
}

func assertBadRequest(t *testing.T, client *Client[googleDirectionsResponse]) {
	t.Helper()
	entity, err := GetForEntity[string](context.Background(), client, googleDirectionsRestURI)
	if err != nil {
		t.Fatal(err)
	}
	if !entity.IsFailure() {
		t.Error("Although the certificate was trustworthy, the request is supposed to return an error.")
	}
	if failure, err := entity.GetError(); err != nil || failure == nil {
		t.Errorf("GetError() = %v, %v", failure, err)
	}
	if entity.HTTPStatus() != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", entity.HTTPStatus(), http.StatusBadRequest)
	}
}

func TestSSL_OfficiallyTrustedTarget_SucceedsWithBadRequest(t *testing.T) {
	client, err := New[googleDirectionsResponse](WithUnknownFieldsAllowed())
	if err != nil {
		t.Fatal(err)
	}
	assertBadRequest(t, client)
}

func TestSSL_OfficiallyTrustedTargetAndCustomCertificate_SucceedsWithBadRequest(t *testing.T) {
	client, err := New[googleDirectionsResponse](
		WithUnknownFieldsAllowed(), WithTrustedCertificates(certificatePath))
	if err != nil {
		t.Fatal(err)
	}
	assertBadRequest(t, client)
}
