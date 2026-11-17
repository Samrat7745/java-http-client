// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

package httpclient

import "testing"

func TestBearerToken_GivenToken_ReturnsCorrectHeader(t *testing.T) {
	token := "aRandomToken123"
	h := BearerToken(token)
	if h.Name != "Authorization" {
		t.Errorf("incorrect authorization header type: %q", h.Name)
	}
	if want := "Bearer " + token; h.Value != want {
		t.Errorf("incorrect authorization header value: got %q, want %q", h.Value, want)
	}
}
