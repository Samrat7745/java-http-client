// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

package httpclient

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func assertErrorPrefix(t *testing.T, err error, prefix string) {
	t.Helper()
	var csErr *Error
	if !errors.As(err, &csErr) {
		t.Fatalf("expected *Error, got %T (%v)", err, err)
	}
	if !strings.HasPrefix(csErr.Error(), prefix) {
		t.Errorf("error %q does not start with %q", csErr.Error(), prefix)
	}
}

func TestHandleResponse_SuccessWithValidBody(t *testing.T) {
	content := "-4521986347761239845"
	entity, err := handleResponse[string, string](http.StatusOK, []byte(content), true)
	if err != nil {
		t.Fatal(err)
	}
	if !entity.IsSuccess() || entity.IsFailure() {
		t.Error("expected successful entity")
	}
	if entity.HTTPStatus() != http.StatusOK {
		t.Errorf("status = %d", entity.HTTPStatus())
	}
	got, err := entity.Get()
	if err != nil || got == nil || *got != content {
		t.Errorf("Get() = %v, %v; want %q", got, err, content)
	}
	_, err = entity.GetError()
	assertErrorPrefix(t, err, "Expected failure but response was successful with status")
}

func TestHandleResponse_SuccessWithoutContent(t *testing.T) {
	for name, body := range map[string][]byte{"nil entity": nil, "empty entity": {}} {
		t.Run(name, func(t *testing.T) {
			entity, err := handleResponse[string, string](http.StatusOK, body, true)
			if err != nil {
				t.Fatal(err)
			}
			if !entity.IsSuccess() {
				t.Error("expected successful entity")
			}
			if entity.HTTPStatus() != http.StatusOK {
				t.Errorf("status = %d", entity.HTTPStatus())
			}
			if got, err := entity.Get(); err != nil || got != nil {
				t.Errorf("Get() = %v, %v; want nil, nil", got, err)
			}
		})
	}
}

func TestHandleResponse_FailureCodeWithBody(t *testing.T) {
	failure := "Something went wrong."
	entity, err := handleResponse[string, string](
		http.StatusInternalServerError, []byte(failure), true)
	if err != nil {
		t.Fatal(err)
	}
	if !entity.IsFailure() || entity.IsSuccess() {
		t.Error("expected failed entity")
	}
	if entity.HTTPStatus() != http.StatusInternalServerError {
		t.Errorf("status = %d", entity.HTTPStatus())
	}
	got, err := entity.GetError()
	if err != nil || got == nil || *got != failure {
		t.Errorf("GetError() = %v, %v; want %q", got, err, failure)
	}
	_, err = entity.Get()
	assertErrorPrefix(t, err, "Request has failed with status")
}

func TestHandleResponse_FailureCodeWithoutContent(t *testing.T) {
	for name, body := range map[string][]byte{"nil entity": nil, "empty entity": {}} {
		t.Run(name, func(t *testing.T) {
			entity, err := handleResponse[string, string](http.StatusContinue, body, true)
			if err != nil {
				t.Fatal(err)
			}
			if !entity.IsFailure() {
				t.Error("expected failed entity")
			}
			if entity.HTTPStatus() != http.StatusContinue {
				t.Errorf("status = %d", entity.HTTPStatus())
			}
			if got, err := entity.GetError(); err != nil || got != nil {
				t.Errorf("GetError() = %v, %v; want nil, nil", got, err)
			}
		})
	}
}

type payload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestHandleResponse_DecodesJSONForBothSides(t *testing.T) {
	body := []byte(`{"name":"a","count":2}`)
	ok, err := handleResponse[payload, payload](http.StatusCreated, body, true)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := ok.Get(); got == nil || *got != (payload{"a", 2}) {
		t.Errorf("Get() = %v", got)
	}
	failed, err := handleResponse[payload, payload](http.StatusBadRequest, body, true)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := failed.GetError(); got == nil || *got != (payload{"a", 2}) {
		t.Errorf("GetError() = %v", got)
	}
}

func TestHandleResponse_UnknownFields(t *testing.T) {
	body := []byte(`{"name":"a","extra":true}`)
	if _, err := handleResponse[string, payload](http.StatusOK, body, true); err == nil {
		t.Error("expected unknown field to be rejected in strict mode")
	}
	entity, err := handleResponse[string, payload](http.StatusOK, body, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := entity.Get(); got == nil || got.Name != "a" {
		t.Errorf("Get() = %v", got)
	}
}

func TestHandleResponse_StringSuccessType(t *testing.T) {
	for body, want := range map[string]string{`"text"`: "text", `12.5`: "12.5", `true`: "true"} {
		entity, err := handleResponse[string, string](http.StatusOK, []byte(body), true)
		if err != nil {
			t.Fatalf("%s: %v", body, err)
		}
		if got, _ := entity.Get(); got == nil || *got != want {
			t.Errorf("%s: Get() = %v, want %q", body, got, want)
		}
	}
	entity, err := handleResponse[string, string](http.StatusOK, []byte(`null`), true)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := entity.Get(); got != nil {
		t.Errorf("null: Get() = %q, want nil", *got)
	}
	for _, body := range []string{`{"a":1}`, `[1]`, `not json`} {
		if _, err := handleResponse[string, string](http.StatusOK, []byte(body), true); err == nil {
			t.Errorf("%s: expected error", body)
		}
	}
}

func TestHandleResponse_InvalidFailureBody(t *testing.T) {
	_, err := handleResponse[payload, string](http.StatusBadGateway, []byte("<html>"), true)
	if err == nil {
		t.Error("expected decoding error")
	}
}
