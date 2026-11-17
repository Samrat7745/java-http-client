// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// ResponseEntity is the outcome of a request: the HTTP status code together
// with either the success content of type R or the failure content of type L.
//
// Content is held by pointer; a nil pointer means the response had no body.
type ResponseEntity[L, R any] struct {
	httpStatus int
	success    bool
	content    *R
	failure    *L
}

// Success returns a successful entity. content may be nil.
func Success[L, R any](httpStatus int, content *R) ResponseEntity[L, R] {
	return ResponseEntity[L, R]{httpStatus: httpStatus, success: true, content: content}
}

// Failed returns a failed entity. failure may be nil.
func Failed[L, R any](httpStatus int, failure *L) ResponseEntity[L, R] {
	return ResponseEntity[L, R]{httpStatus: httpStatus, failure: failure}
}

// HTTPStatus returns the HTTP status code of the response.
func (e ResponseEntity[L, R]) HTTPStatus() int { return e.httpStatus }

// IsSuccess reports whether the request succeeded (2xx status code).
func (e ResponseEntity[L, R]) IsSuccess() bool { return e.success }

// IsFailure reports whether the request failed.
func (e ResponseEntity[L, R]) IsFailure() bool { return !e.success }

// Get returns the success content, or an [*Error] if the request failed.
func (e ResponseEntity[L, R]) Get() (*R, error) {
	if !e.success {
		return nil, newError(fmt.Sprintf(
			"Request has failed with status <%d>: %s", e.httpStatus, describe(e.failure)), nil)
	}
	return e.content, nil
}

// GetError returns the failure content, or an [*Error] if the request
// succeeded.
func (e ResponseEntity[L, R]) GetError() (*L, error) {
	if e.success {
		return nil, newError(fmt.Sprintf(
			"Expected failure but response was successful with status <%d>: %s",
			e.httpStatus, describe(e.content)), nil)
	}
	return e.failure, nil
}

func describe[T any](v *T) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprintf("%v", *v)
}

// handleResponse maps a status code and response body to a ResponseEntity.
// Status codes 200-299 are successes whose body is decoded as R; everything
// else is a failure whose body is decoded as L. Empty bodies yield nil content.
// If strict is set, JSON object fields unknown to the target type are rejected.
func handleResponse[L, R any](status int, body []byte, strict bool) (ResponseEntity[L, R], error) {
	if isSuccess(status) {
		content, err := decode[R](body, strict)
		if err != nil {
			return ResponseEntity[L, R]{}, err
		}
		return Success[L](status, content), nil
	}
	failure, err := decodeFailure[L](body, strict)
	if err != nil {
		return ResponseEntity[L, R]{}, err
	}
	return Failed[L, R](status, failure), nil
}

func isSuccess(status int) bool { return status >= 200 && status < 300 }

// decodeFailure decodes a failure body. If the failure type is string the body
// is returned verbatim, otherwise it is decoded as JSON.
func decodeFailure[L any](body []byte, strict bool) (*L, error) {
	if len(body) == 0 {
		return nil, nil
	}
	var failure L
	if s, ok := any(&failure).(*string); ok {
		*s = string(body)
		return &failure, nil
	}
	return decode[L](body, strict)
}

// decode decodes a JSON body. An empty body and a JSON null both yield nil.
func decode[T any](body []byte, strict bool) (*T, error) {
	if len(body) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	var v *T
	if sp, ok := any(&v).(**string); ok {
		// JSON scalars are accepted for string targets, so that a bare
		// number or boolean body can be read as its literal text.
		dec.UseNumber()
		var raw any
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		var s string
		switch x := raw.(type) {
		case nil:
			return nil, nil
		case string:
			s = x
		case json.Number:
			s = x.String()
		case bool:
			s = strconv.FormatBool(x)
		default:
			return nil, fmt.Errorf("json: cannot decode %T into a string", raw)
		}
		*sp = &s
		return v, nil
	}
	if strict {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}
