// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

const (
	dummyURL     = "https://cs-virtual-cloud.com/"
	dummyBody    = "payload"
	dummyContent = "response"
)

// fakeTransport answers every request with a canned response or error and
// records what it was asked to send.
type fakeTransport struct {
	status int
	body   string
	err    error
	calls  int
	last   *http.Request
	sent   string
}

func (f *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	f.last = req
	if req.Body != nil {
		data, _ := io.ReadAll(req.Body)
		f.sent = string(data)
	}
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{
		StatusCode: f.status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Request:    req,
	}, nil
}

func newTestClient(t *testing.T, transport http.RoundTripper, opts ...Option) *Client[string] {
	t.Helper()
	client, err := New[string](append(opts, withTransport(transport))...)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func assertCalls(t *testing.T, f *fakeTransport, method string) {
	t.Helper()
	if f.calls != 1 {
		t.Errorf("expected exactly 1 request, got %d", f.calls)
	}
	if f.last != nil && f.last.Method != method {
		t.Errorf("method = %s, want %s", f.last.Method, method)
	}
}

func assertWrapped(t *testing.T, err error, message string, cause error) {
	t.Helper()
	var csErr *Error
	if !errors.As(err, &csErr) {
		t.Fatalf("expected *Error, got %T (%v)", err, err)
	}
	if csErr.Message() != message {
		t.Errorf("message = %q, want %q", csErr.Message(), message)
	}
	if !errors.Is(err, cause) {
		t.Errorf("error %v does not wrap cause %v", err, cause)
	}
}

func TestGetForObject_ThrowingRequest_ReturnsErrorWithCause(t *testing.T) {
	cause := errors.New("exception message")
	f := &fakeTransport{err: cause}
	_, err := GetForObject[string](context.Background(), newTestClient(t, f), dummyURL)
	assertWrapped(t, err, "GET request failed.", cause)
	assertCalls(t, f, http.MethodGet)
}

func TestGetForObject_SuccessfulRequest_ReturnsContent(t *testing.T) {
	f := &fakeTransport{status: http.StatusOK, body: strconv.Quote(dummyContent)}
	got, err := GetForObject[string](context.Background(), newTestClient(t, f), dummyURL)
	if err != nil || got == nil || *got != dummyContent {
		t.Errorf("got %v, %v; want %q", got, err, dummyContent)
	}
	assertCalls(t, f, http.MethodGet)
}

func TestGetForObject_InternalServerError_ReturnsErrorWithMessage(t *testing.T) {
	f := &fakeTransport{status: http.StatusInternalServerError, body: dummyContent}
	_, err := GetForObject[string](context.Background(), newTestClient(t, f), dummyURL)
	var csErr *Error
	if !errors.As(err, &csErr) {
		t.Fatalf("expected *Error, got %T (%v)", err, err)
	}
	for _, part := range []string{"Request has failed", "500", dummyContent} {
		if !strings.Contains(csErr.Error(), part) {
			t.Errorf("error %q does not contain %q", csErr.Error(), part)
		}
	}
	assertCalls(t, f, http.MethodGet)
}

func TestGetForEntity_SuccessfulRequest_ReturnsContent(t *testing.T) {
	f := &fakeTransport{status: http.StatusOK, body: strconv.Quote(dummyContent)}
	entity, err := GetForEntity[string](context.Background(), newTestClient(t, f), dummyURL)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := entity.Get(); !entity.IsSuccess() || entity.HTTPStatus() != http.StatusOK ||
		got == nil || *got != dummyContent {
		t.Errorf("unexpected entity %+v", entity)
	}
	assertCalls(t, f, http.MethodGet)
}

func TestGetForEntity_ThrowingRequest_ReturnsErrorWithCause(t *testing.T) {
	cause := errors.New("exception message")
	f := &fakeTransport{err: cause}
	_, err := GetForEntity[string](context.Background(), newTestClient(t, f), dummyURL)
	assertWrapped(t, err, "GET request failed.", cause)
	assertCalls(t, f, http.MethodGet)
}

func TestGetForEntity_FailureResponse_IsNotAnError(t *testing.T) {
	f := &fakeTransport{status: http.StatusNotFound, body: "missing"}
	entity, err := GetForEntity[string](context.Background(), newTestClient(t, f), dummyURL)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := entity.GetError(); !entity.IsFailure() || got == nil || *got != "missing" ||
		entity.HTTPStatus() != http.StatusNotFound {
		t.Errorf("unexpected entity %+v", entity)
	}
}

func TestPostForEntity_ThrowingRequest_ReturnsErrorWithCause(t *testing.T) {
	cause := errors.New("exception message")
	f := &fakeTransport{err: cause}
	_, err := PostForEntity[string](context.Background(), newTestClient(t, f), dummyURL, dummyBody)
	assertWrapped(t, err, "POST request failed.", cause)
	assertCalls(t, f, http.MethodPost)
}

func TestPostForObject_SuccessfulRequest_ReturnsContent(t *testing.T) {
	f := &fakeTransport{status: http.StatusOK, body: strconv.Quote(dummyContent)}
	got, err := PostForObject[string](
		context.Background(), newTestClient(t, f), dummyURL, dummyBody)
	if err != nil || got == nil || *got != dummyContent {
		t.Errorf("got %v, %v; want %q", got, err, dummyContent)
	}
	assertCalls(t, f, http.MethodPost)
}

func TestPostForEntity_SuccessfulRequest_ReturnsContent(t *testing.T) {
	f := &fakeTransport{status: http.StatusOK, body: strconv.Quote(dummyContent)}
	entity, err := PostForEntity[string](
		context.Background(), newTestClient(t, f), dummyURL, dummyBody)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := entity.Get(); !entity.IsSuccess() || got == nil || *got != dummyContent {
		t.Errorf("unexpected entity %+v", entity)
	}
	assertCalls(t, f, http.MethodPost)
}

func TestPut_ThrowingRequest_ReturnsErrorWithCause(t *testing.T) {
	cause := errors.New("exception message")
	f := &fakeTransport{err: cause}
	err := newTestClient(t, f).Put(context.Background(), dummyURL, dummyBody)
	assertWrapped(t, err, "PUT request failed.", cause)
	assertCalls(t, f, http.MethodPut)
}

func TestPut_SuccessfulRequest_EmitsPutRequest(t *testing.T) {
	f := &fakeTransport{status: http.StatusOK}
	if err := newTestClient(t, f).Put(context.Background(), dummyURL, dummyBody); err != nil {
		t.Fatal(err)
	}
	assertCalls(t, f, http.MethodPut)
}

func assertFailureCause(t *testing.T, err error, message string) {
	t.Helper()
	var csErr *Error
	if !errors.As(err, &csErr) {
		t.Fatalf("expected *Error, got %T (%v)", err, err)
	}
	if csErr.Message() != message {
		t.Errorf("message = %q, want %q", csErr.Message(), message)
	}
	var cause *Error
	if !errors.As(csErr.Unwrap(), &cause) || !strings.Contains(cause.Error(), "500") {
		t.Errorf("cause %v does not describe status 500", csErr.Unwrap())
	}
}

func TestPut_InternalServerError_ReturnsErrorWithMessage(t *testing.T) {
	f := &fakeTransport{status: http.StatusInternalServerError}
	err := newTestClient(t, f).Put(context.Background(), dummyURL, dummyBody)
	assertFailureCause(t, err, "PUT request failed.")
	assertCalls(t, f, http.MethodPut)
}

func TestDelete_ThrowingRequest_ReturnsErrorWithCause(t *testing.T) {
	cause := errors.New("exception message")
	f := &fakeTransport{err: cause}
	err := newTestClient(t, f).Delete(context.Background(), dummyURL)
	assertWrapped(t, err, "DELETE request failed.", cause)
	assertCalls(t, f, http.MethodDelete)
}

func TestDelete_SuccessfulRequest_EmitsDeleteRequest(t *testing.T) {
	f := &fakeTransport{status: http.StatusOK}
	if err := newTestClient(t, f).Delete(context.Background(), dummyURL); err != nil {
		t.Fatal(err)
	}
	assertCalls(t, f, http.MethodDelete)
}

func TestDelete_InternalServerError_ReturnsErrorWithMessage(t *testing.T) {
	f := &fakeTransport{status: http.StatusInternalServerError}
	err := newTestClient(t, f).Delete(context.Background(), dummyURL)
	assertFailureCause(t, err, "DELETE request failed.")
	assertCalls(t, f, http.MethodDelete)
}

func TestRequestShape(t *testing.T) {
	ctx := context.Background()
	auth := BearerToken("t0k3n")

	t.Run("POST sends JSON body and headers", func(t *testing.T) {
		f := &fakeTransport{status: http.StatusOK}
		body := map[string]string{"k": "<v&>"}
		if _, err := PostForEntity[string](ctx, newTestClient(t, f), dummyURL, body, auth); err != nil {
			t.Fatal(err)
		}
		if f.sent != `{"k":"<v&>"}` {
			t.Errorf("body = %s", f.sent)
		}
		for name, want := range map[string]string{
			"Accept": "application/json", "Content-Type": "application/json",
			"Authorization": "Bearer t0k3n",
		} {
			if got := f.last.Header.Get(name); got != want {
				t.Errorf("header %s = %q, want %q", name, got, want)
			}
		}
	})
	t.Run("PUT sends JSON body, nil becomes null", func(t *testing.T) {
		f := &fakeTransport{status: http.StatusNoContent}
		if err := newTestClient(t, f).Put(ctx, dummyURL, nil); err != nil {
			t.Fatal(err)
		}
		if f.sent != "null" || f.last.Header.Get("Content-Type") != "application/json" {
			t.Errorf("body = %q, headers = %v", f.sent, f.last.Header)
		}
	})
	t.Run("GET and DELETE send no body and no JSON headers", func(t *testing.T) {
		f := &fakeTransport{status: http.StatusOK}
		client := newTestClient(t, f)
		if _, err := GetForEntity[string](ctx, client, dummyURL, auth); err != nil {
			t.Fatal(err)
		}
		if f.last.Body != nil || f.last.Header.Get("Accept") != "" ||
			f.last.Header.Get("Authorization") != "Bearer t0k3n" {
			t.Errorf("unexpected GET request: %v", f.last.Header)
		}
		if err := client.Delete(ctx, dummyURL); err != nil {
			t.Fatal(err)
		}
		if f.last.Body != nil || f.last.Header.Get("Content-Type") != "" {
			t.Errorf("unexpected DELETE request: %v", f.last.Header)
		}
	})
	t.Run("unencodable body fails the request", func(t *testing.T) {
		f := &fakeTransport{status: http.StatusOK}
		_, err := PostForEntity[string](ctx, newTestClient(t, f), dummyURL, make(chan int))
		var csErr *Error
		if !errors.As(err, &csErr) || csErr.Message() != "POST request failed." || f.calls != 0 {
			t.Errorf("err = %v, calls = %d", err, f.calls)
		}
	})
	t.Run("invalid URL fails the request", func(t *testing.T) {
		f := &fakeTransport{status: http.StatusOK}
		_, err := GetForEntity[string](ctx, newTestClient(t, f), "://nope")
		var csErr *Error
		if !errors.As(err, &csErr) || f.calls != 0 {
			t.Errorf("err = %v, calls = %d", err, f.calls)
		}
	})
}

func TestPutAndDelete_StringFailureBodyIsReported(t *testing.T) {
	f := &fakeTransport{status: http.StatusConflict, body: "already there"}
	err := newTestClient(t, f).Put(context.Background(), dummyURL, dummyBody)
	if err == nil || !strings.Contains(err.Error(), "<409>: already there") {
		t.Errorf("err = %v", err)
	}
}

func TestConstructors(t *testing.T) {
	if c, err := NewDefault(); err != nil || c == nil {
		t.Errorf("NewDefault() = %v, %v", c, err)
	}
	if c, err := NewDefaultWithRetries(2); err != nil || c == nil {
		t.Errorf("NewDefaultWithRetries() = %v, %v", c, err)
	}
	c, err := NewWithRetries[payload](-3)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.httpClient.Transport.(*retryTransport); ok {
		t.Error("negative number of retries must be treated as 0")
	}
	c.Close()
}

// flakyTransport fails the first failures requests with err.
type flakyTransport struct {
	failures int
	err      error
	calls    int
}

func (f *flakyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	if req.Body != nil {
		_, _ = io.Copy(io.Discard, req.Body)
	}
	if f.calls <= f.failures {
		return nil, f.err
	}
	return &http.Response{
		StatusCode: http.StatusOK, Header: http.Header{},
		Body: io.NopCloser(strings.NewReader("")), Request: req,
	}, nil
}

func TestRetries(t *testing.T) {
	ctx := context.Background()
	t.Run("GET is retried until it succeeds", func(t *testing.T) {
		f := &flakyTransport{failures: 2, err: io.ErrUnexpectedEOF}
		if _, err := GetForEntity[string](ctx, newTestClient(t, f, WithRetries(2)), dummyURL); err != nil {
			t.Fatal(err)
		}
		if f.calls != 3 {
			t.Errorf("calls = %d, want 3", f.calls)
		}
	})
	t.Run("GET gives up after the configured retries", func(t *testing.T) {
		f := &flakyTransport{failures: 10, err: io.ErrUnexpectedEOF}
		_, err := GetForEntity[string](ctx, newTestClient(t, f, WithRetries(2)), dummyURL)
		if !errors.Is(err, io.ErrUnexpectedEOF) || f.calls != 3 {
			t.Errorf("err = %v, calls = %d", err, f.calls)
		}
	})
	t.Run("DELETE is retried", func(t *testing.T) {
		f := &flakyTransport{failures: 1, err: io.ErrUnexpectedEOF}
		if err := newTestClient(t, f, WithRetries(1)).Delete(ctx, dummyURL); err != nil {
			t.Fatal(err)
		}
		if f.calls != 2 {
			t.Errorf("calls = %d, want 2", f.calls)
		}
	})
	t.Run("requests with a body are not retried", func(t *testing.T) {
		f := &flakyTransport{failures: 1, err: io.ErrUnexpectedEOF}
		if err := newTestClient(t, f, WithRetries(3)).Put(ctx, dummyURL, dummyBody); err == nil {
			t.Error("expected error")
		}
		if f.calls != 1 {
			t.Errorf("calls = %d, want 1", f.calls)
		}
	})
	t.Run("no retries by default", func(t *testing.T) {
		f := &flakyTransport{failures: 1, err: io.ErrUnexpectedEOF}
		if _, err := GetForEntity[string](ctx, newTestClient(t, f), dummyURL); err == nil {
			t.Error("expected error")
		}
		if f.calls != 1 {
			t.Errorf("calls = %d, want 1", f.calls)
		}
	})
	t.Run("cancellation is not retried", func(t *testing.T) {
		f := &flakyTransport{failures: 10, err: context.DeadlineExceeded}
		if _, err := GetForEntity[string](ctx, newTestClient(t, f, WithRetries(3)), dummyURL); err == nil {
			t.Error("expected error")
		}
		if f.calls != 1 {
			t.Errorf("calls = %d, want 1", f.calls)
		}
	})
}

func TestRetries_ConnectionRefusedIsNotRetried(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	client, err := NewDefaultWithRetries(3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GetForEntity[string](context.Background(), client, url); err == nil {
		t.Error("expected error")
	}
}

func TestRedirects(t *testing.T) {
	var targetHits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/target", func(w http.ResponseWriter, r *http.Request) {
		targetHits.Add(1)
		_, _ = w.Write([]byte(strconv.Quote(r.Method)))
	})
	mux.HandleFunc("/found", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/target", http.StatusFound)
	})
	mux.HandleFunc("/see-other", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/target", http.StatusSeeOther)
	})
	mux.HandleFunc("/loop", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client, err := NewDefault()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx := context.Background()

	got, err := GetForObject[string](ctx, client, srv.URL+"/found")
	if err != nil || got == nil || *got != http.MethodGet {
		t.Errorf("GET redirect: got %v, %v", got, err)
	}

	targetHits.Store(0)
	entity, err := PostForEntity[string](ctx, client, srv.URL+"/found", dummyBody)
	if err != nil {
		t.Fatal(err)
	}
	if !entity.IsFailure() || entity.HTTPStatus() != http.StatusFound || targetHits.Load() != 0 {
		t.Errorf("POST must not follow a 302: %+v, hits = %d", entity, targetHits.Load())
	}

	got, err = PostForObject[string](ctx, client, srv.URL+"/see-other", dummyBody)
	if err != nil || got == nil || *got != http.MethodGet {
		t.Errorf("POST 303 redirect: got %v, %v", got, err)
	}

	if _, err := GetForEntity[string](ctx, client, srv.URL+"/loop"); err == nil ||
		!strings.Contains(err.Error(), "maximum redirects") {
		t.Errorf("redirect loop: err = %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()
	client, err := NewDefault()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = GetForEntity[string](ctx, client, srv.URL)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
