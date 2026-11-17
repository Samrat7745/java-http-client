// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

package httpclient

// Error is returned when a client operation fails.
//
// Use [errors.As] to detect it and [errors.Is] / [errors.As] to inspect the
// underlying cause.
type Error struct {
	message string
	cause   error
}

func newError(message string, cause error) *Error {
	return &Error{message: message, cause: cause}
}

// Error returns the message followed by the message of the cause, if any.
func (e *Error) Error() string {
	if e.cause == nil {
		return e.message
	}
	return e.message + " " + e.cause.Error()
}

// Message returns the detail message without the cause.
func (e *Error) Message() string { return e.message }

// Unwrap returns the cause of the error, or nil.
func (e *Error) Unwrap() error { return e.cause }
