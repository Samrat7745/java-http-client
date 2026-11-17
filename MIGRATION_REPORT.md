# Java to Go Migration Report

This report describes the migration of the Carbyne Stack HTTP client library
from Java (release `0.2.0`) to Go on the branch `feat/go-migration`: what was
changed, how the API maps, and where the behaviour differs.

## Summary

- The library is now a single Go package, `httpclient`, in the repository
  root. It uses the standard library only.
- All Java sources, the Maven build and the Java-specific third-party license
  files were removed.
- The module path is `github.com/Samrat7745/java-http-client` and the
  release tag format `java-http-client-vX.Y.Z` is unchanged.
- The Java test suite was run before removal and passed. Every Java test has a
  Go counterpart, and all Go tests pass.
- Most behaviour was preserved deliberately. The differences are listed in
  [Behaviour changes](#behaviour-changes); the ones most likely to affect
  users are the request encoding (now UTF-8), connection reuse, the removal of
  JVM system property support, and retries of requests with a body.

## What changed in the repository

### Removed

| Path                                        | Reason                                    |
| ------------------------------------------- | ----------------------------------------- |
| `src/main/java/**` (9 classes)              | Replaced by the Go package                |
| `src/test/java/**` (5 test classes)         | Replaced by Go tests                      |
| `src/test/resources/keyStore.jks`           | Go reads the PEM key pair instead         |
| `pom.xml`                                   | Replaced by `go.mod`                      |
| `.github/templates/sbom.ftl`                | Template of the Maven license plugin      |
| `3RD-PARTY-LICENSES/<java dependency>/**`   | The Go module has no third-party dependencies |

### Added

| Path                                  | Content                                           |
| ------------------------------------- | ------------------------------------------------- |
| `go.mod`                              | Module definition, Go 1.26 minimum, no dependencies |
| `doc.go`                              | Package documentation                             |
| `client.go`                           | `Client`, options, request functions, redirect policy |
| `response.go`                         | `ResponseEntity` and response decoding            |
| `errors.go`                           | `Error`                                           |
| `bearer.go`                           | `Header`, `BearerToken`                           |
| `tls.go`                              | TLS configuration and certificate loading         |
| `retry.go`                            | Retry transport                                   |
| `*_test.go`                           | Unit tests, and integration tests behind a build tag |
| `MIGRATION_REPORT.md`                 | This report                                       |

### Moved

`src/test/resources/cli_test.pem` and `cli_test.key` moved to `testdata/`,
unchanged.

### Maintenance files

| File                                              | Change                                         |
| ------------------------------------------------- | ---------------------------------------------- |
| `.github/workflows/java.build-and-test.yaml`      | Renamed to `go.build-and-test.yaml`. Same triggers and `build` / `test` jobs; the JDK 8 / 17 matrix became Go 1.26 / 1.27. Runs `gofmt`, `go build`, `go vet` and `go test -race -tags integration` and uploads coverage to Codecov. |
| `.github/workflows/java.publish.yaml`             | Renamed to `go.publish.yaml`. Same tag trigger. Keeps the upload of license obligation resources to the release; the Maven deploy to GitHub Packages was dropped because a Go module is consumed from the repository. |
| `release-please-config.json`                      | `release-type` changed from `maven` to `go`.   |
| `.pre-commit-config.yaml`                         | Java formatter hook replaced by the Go formatter hook of the same hook repository. |
| `.gitignore`                                      | Go entries added; existing entries kept.       |
| `README.md`                                       | Describes the Go library, with usage and development instructions. Badges, license and contributing sections kept. |
| `3RD-PARTY-LICENSES/sbom.xml`                     | Now an empty dependency list.                  |

Unchanged: `LICENSE`, `NOTICE.md`, `CODEOWNERS`, `CODE_OF_CONDUCT.md`,
`CONTRIBUTING.md`, `.codacy.yaml`, `commitlint.config.js`,
`.github/workflows/release.yaml`, `auto-labeler.yml`,
`stale-issue-cleanup.yml`.

`CHANGELOG.md` and `.release-please-manifest.json` were not edited by hand.
They are maintained by release-please, which will create version `0.3.0` from
the breaking-change commit of this migration.

## API mapping

| Java                                                   | Go                                                    |
| ------------------------------------------------------ | ----------------------------------------------------- |
| `CsHttpClient<L>`                                      | `Client[L]`                                           |
| `CsHttpClient.<L>builder()...build()`                  | `New[L](opts...)`                                     |
| `withNumberOfRetries(n)`                               | `WithRetries(n)`                                      |
| `withoutSslValidation(b)`                              | `WithoutSSLValidation(b)`                             |
| `withTrustedCertificates(List<File>)`                  | `WithTrustedCertificates(files...)`                   |
| `withFailureType(Class)`                               | Type argument `L` of `New`                            |
| `createDefault()`                                      | `NewDefault()`                                        |
| `createDefaultWithRetries(n)`                          | `NewDefaultWithRetries(n)`                            |
| `createWithRetries(n, type)`                           | `NewWithRetries[L](n)`                                |
| `client.getForObject(url, [headers,] type)`            | `GetForObject[R](ctx, client, url, headers...)`       |
| `client.getForEntity(url, [headers,] type)`            | `GetForEntity[R](ctx, client, url, headers...)`       |
| `client.postForObject(url, [headers,] body, type)`     | `PostForObject[R](ctx, client, url, body, headers...)` |
| `client.postForEntity(url, [headers,] body, type)`     | `PostForEntity[R](ctx, client, url, body, headers...)` |
| `client.put(url, [headers,] body)`                     | `client.Put(ctx, url, body, headers...)`              |
| `client.delete(url, [headers])`                        | `client.Delete(ctx, url, headers...)`                 |
| `CsResponseEntity<L, R>`                               | `ResponseEntity[L, R]`                                |
| `CsResponseEntity.success` / `failed`                  | `Success` / `Failed`                                  |
| `getHttpStatus()`, `isSuccess()`, `isFailure()`        | `HTTPStatus()`, `IsSuccess()`, `IsFailure()`          |
| `get()`, `getError()`                                  | `Get()`, `GetError()`                                 |
| `getContent()` (vavr `Either`)                         | No equivalent; use `IsSuccess()` with `Get()` / `GetError()` |
| `CsHttpClientException`                                | `*Error` (`Message()`, `Unwrap()`)                    |
| `BearerTokenUtils.createBearerToken(token)`            | `BearerToken(token)`                                  |
| `org.apache.http.Header`                               | `Header{Name, Value}`                                 |
| `CsResponseHandler`                                    | Internal (`handleResponse`)                           |
| `HttpClientFactory`                                    | Internal (transport injection for tests)              |
| `CompositeX509TrustManager`, `X509TrustManagerUtils`, `ExtendedSSLContextBuilder` | Internal (`newCertPool`): one certificate pool holding the system roots and the given certificates |

New in Go, without a Java counterpart:

- `WithUnknownFieldsAllowed()` replaces Jackson's
  `@JsonIgnoreProperties(ignoreUnknown = true)` on a response type.
- `WithLogger(*slog.Logger)` selects the logger for debug output.
- `Client.Close()` closes idle pooled connections.
- Every request takes a `context.Context` for cancellation and deadlines.

### Differences in API shape

- **Typed requests are functions, not methods.** Go methods cannot have their
  own type parameters, so `GetForObject`, `GetForEntity`, `PostForObject` and
  `PostForEntity` are package-level functions that take the client as an
  argument. Only the response type has to be written out; the failure type is
  inferred from the client.
- **Errors are returned, not thrown.** Each operation returns an `error`. Use
  `errors.As(err, &csErr)` for `*Error` and `errors.Is` / `errors.As` for the
  cause.
- **`null` is a nil pointer.** `Get()` and `GetError()` return `*R` and `*L`.
  A response without a body yields `nil`, as it yielded `null` in Java.
- **URLs are strings** rather than `java.net.URI`. An unparsable URL fails the
  request with an `*Error`.
- **Constructors return an error.** `createWithRetries` logged and returned
  `null` if the client could not be built; the Go constructors return the
  error.
- **`ResponseEntity` is not serializable.** `java.io.Serializable` has no
  counterpart.

## Behaviour changes

### Preserved on purpose

These would have changed with a naive port and were implemented explicitly.

- **Success and failure mapping.** Status codes 200-299 are successes decoded
  as `R`; every other status is a failure decoded as `L`. An absent or empty
  body yields nil content.
- **String failure type.** If `L` is `string`, the failure body is used
  verbatim and not parsed as JSON.
- **String success type.** If `R` is `string`, a JSON string is unquoted and a
  bare JSON number or boolean is returned as its literal text, as Jackson did.
  `encoding/json` alone would reject the latter.
- **Unknown JSON fields are rejected** by default, matching Jackson's default.
  `encoding/json` alone would ignore them.
- **Error messages.** `Request has failed with status <500>: ...`,
  `Expected failure but response was successful with status <200>: ...`,
  `GET request failed.` (and POST / PUT / DELETE), and
  `Unable to configure certificate management` are unchanged and available
  through `Error.Message()`.
- **Error nesting of `Put` and `Delete`.** A failure response is reported as
  `PUT request failed.` with the status error as its cause, as in Java.
  `GetForObject` and `PostForObject` return the status error directly.
- **Request headers.** POST and PUT send `Accept: application/json` and
  `Content-Type: application/json`; GET and DELETE send neither. Caller
  headers are added, not replaced.
- **Request body.** `<`, `>` and `&` are not escaped in the JSON body, as with
  Jackson. A nil body is sent as `null`.
- **Trust.** With trusted certificates, a server is accepted if either the
  system roots or the given certificates vouch for it. Disabling validation
  also disables host name verification and takes precedence over trusted
  certificates.
- **Retries.** Go has no retry handler, so one was written after Apache's
  `DefaultHttpRequestRetryHandler(n, false)`: at most `n` retries, only for
  transport errors, and never for timeouts, cancellations, unknown hosts,
  connection failures or TLS errors. A negative count is treated as 0.
- **Redirects.** Only GET requests follow redirects, plus `303 See Other` for
  any method, up to 50 hops. A `301` / `302` / `307` answer to POST, PUT or
  DELETE is returned as a failure response. Go's default would have followed
  them.
- **Response bodies of `Put` and `Delete`** are ignored on success.

### Changed

| Area | Java | Go | Impact |
| ---- | ---- | -- | ------ |
| Request encoding | `StringEntity` encoded the JSON body as ISO-8859-1; characters outside Latin-1 were sent as `?` | UTF-8 | Fixes corrupted non-Latin-1 payloads. Bodies with Latin-1 characters above ASCII are now different bytes on the wire. |
| Response encoding | ISO-8859-1 unless the `Content-Type` named a charset | Bytes are taken as UTF-8 | Correct for JSON. A service answering in another charset is decoded differently. |
| Connections | A new HTTP client, and so a new connection, per request | One pooled transport per `Client`; connections are reused | Faster. Clients should be reused, and `Close()` releases idle connections. |
| Retries of POST / PUT | Retried if the failure happened before the request was fully sent | Never retried | Go cannot tell whether a request was sent. Fewer retries for requests with a body. |
| Retries on stale connections | Not applicable (no reuse) | Go's transport may additionally re-send a body-less request if a reused connection turns out to be closed | Transparent. |
| JVM system properties | `http.proxyHost`, `http.nonProxyHosts`, `javax.net.ssl.trustStore`, `javax.net.ssl.keyStore`, `http.agent` and others were honoured | Proxy from `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`; system roots from the OS, `SSL_CERT_FILE`, `SSL_CERT_DIR` | Deployments configured through `-D` flags need the environment variables instead. Client certificates through `javax.net.ssl.keyStore` are no longer possible. |
| Timeouts | None by default | 30 s to connect and 10 s for the TLS handshake (Go defaults); no overall timeout unless the context has a deadline | Hung connection attempts now fail. |
| Cancellation | Not possible | Through `context.Context` | New capability. |
| Protocol | HTTP/1.1 | HTTP/2 if the server offers it, HTTP/1.1 otherwise | Transparent. |
| `User-Agent` | `Apache-HttpClient/4.5.14 (Java/...)` | `Go-http-client/1.1` (or `2.0`) | Visible in server logs and to user-agent filters. |
| Certificate files | Only the first certificate of each file was loaded | All certificates of a PEM bundle are loaded; a single DER certificate is accepted too | A bundle now trusts every certificate it contains. |
| Certificate errors | Cause was a `CertificateException` | Cause is the file system or parsing error (`fs.ErrNotExist` for a missing file) | Code inspecting the cause changes. |
| Untrusted server | Cause was `SSLHandshakeException` | Cause is `*tls.CertificateVerificationError` wrapping `x509.UnknownAuthorityError` | Code inspecting the cause changes. |
| Host name verification | Common name accepted as fallback | Subject alternative name required | Certificates without a SAN are rejected. |
| Error text | `getMessage()` returned the message only | `Error()` appends the cause, e.g. `PUT request failed. Request has failed with status <500>: null` | `Message()` returns the Java text. |
| JSON field matching | Case-sensitive, by Jackson annotations and bean properties | By `json` struct tags, case-insensitive | Response types need `json` tags. |
| JSON scalar coercion | Jackson coerced between scalars (for example `"12"` to `int`) | Only the coercion to `string` described above | Stricter for non-string targets. |
| `Void` failure type | Any failure body was accepted and discarded | No direct counterpart; with `L = struct{}` a non-empty, non-object body is a decoding error | Use `string` or `any` as failure type to accept any body. |
| Logging | slf4j, debug level | `log/slog`, debug level | Configure through `slog` or `WithLogger`. |
| Distribution | Maven artifact on GitHub Packages | Go module fetched from the repository | See [Open points](#open-points) on versions. |

## Tests

### Java baseline

Before the Java sources were removed, `mvn verify` was run on JDK 17 with
Maven 3.9.9: 25 unit tests and 6 integration tests, all passed.

### Go

| Java test class                     | Tests | Go counterpart                                         |
| ----------------------------------- | ----- | ------------------------------------------------------ |
| `BearerTokenUtilsTest`              | 1     | `bearer_test.go`                                       |
| `CsResponseHandlerTest`             | 6     | `response_test.go`                                     |
| `CsHttpClientTest`                  | 14    | `client_test.go`                                       |
| `CompositeX509TrustManagerTest`     | 4     | `tls_test.go` (`TestCertPool_*`), with the integration tests covering the system roots |
| `SslValidationIT` (local server)    | 4     | `tls_test.go` (`TestSSL_*`), against an `httptest` TLS server using `testdata/cli_test.pem` |
| `SslValidationIT` (Google endpoint) | 2     | `integration_test.go`, build tag `integration`         |

Mockito mocks of the Apache client were replaced by a fake
`http.RoundTripper`. Tests were added for what the Java suite did not cover:
request headers and body, retries, redirects, cancellation, strict and lenient
decoding, string coercion, and invalid, DER and bundled certificate files.

Results:

| Command                                             | Toolchain | Result                         |
| --------------------------------------------------- | --------- | ------------------------------ |
| `go test -race -tags integration ./...`             | Go 1.27.4 | 41 tests passed, 96.1% coverage |
| `go test ./...`                                     | Go 1.27.4 | 39 tests passed                |
| `go test ./...`                                     | Go 1.26.8 | passed                         |
| `go vet -tags integration ./...`, `gofmt -l .`      | Go 1.27.4 | clean                          |

Not verified: the GitHub workflows and the pre-commit hooks were not executed,
as that needs a push and tools that are not installed locally.

## Open points

- **Release tags and Go versions.** The tag format `java-http-client-vX.Y.Z`
  was kept as requested. Go only recognises tags of the form `vX.Y.Z` as
  module versions, so consumers get a pseudo-version
  (`go get github.com/Samrat7745/java-http-client@java-http-client-v0.3.0`
  resolves to one). Switching to plain `vX.Y.Z` tags needs
  `"include-component-in-tag": false` in `release-please-config.json` and the
  new pattern in `go.publish.yaml`.
- **Module path.** The path still contains `java` because it has to match the
  repository URL. Renaming the repository would change the import path.
- **Fork.** The module path names the fork `Samrat7745/java-http-client`, so
  that other modules can fetch it. It has to be changed back to
  `github.com/carbynestack/java-http-client` before the work is merged
  upstream.
- **`commitlint.config.js`** still enforces the scope `java-http-client`.
- **`NOTICE.md`** was left as is; add the contributors of the migration as
  appropriate.
- **Codecov and Codacy** need no configuration change, but the first Go
  coverage upload will not be comparable with the Java history.
