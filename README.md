# Carbyne Stack HTTP Client

[![codecov](https://codecov.io/gh/carbynestack/java-http-client/branch/master/graph/badge.svg?token=iJ4L3PbGvT)](https://codecov.io/gh/carbynestack/java-http-client)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/819ea9adb8a64ae18beb9065a9b00b82)](https://www.codacy.com?utm_source=github.com&utm_medium=referral&utm_content=carbynestack/java-http-client&utm_campaign=Badge_Grade)
[![Known Vulnerabilities](https://snyk.io/test/github/carbynestack/java-http-client/badge.svg)](https://snyk.io/test/github/carbynestack/java-http-client)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-%23FE5196?logo=conventionalcommits&logoColor=white)](https://conventionalcommits.org)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit&logoColor=white)](https://github.com/pre-commit/pre-commit)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

This project provides common functionality for the Go-based HTTP clients for
the Carbyne Stack microservices.

> **NOTE**: This library has been migrated from Java to Go. The last Java
> release is `0.2.0`. See the [migration report](MIGRATION_REPORT.md) for the
> API mapping and the changes in behaviour.

## Usage

The module requires Go 1.26 or later and has no dependencies besides the
standard library.

```shell
go get github.com/carbynestack/java-http-client
```

```go
import httpclient "github.com/carbynestack/java-http-client"

// ErrorBody is what the service returns for non 2xx status codes.
type ErrorBody struct {
	Message string `json:"message"`
}

client, err := httpclient.New[ErrorBody](
	httpclient.WithRetries(3),
	httpclient.WithTrustedCertificates("/etc/cs/ca.pem"),
)
if err != nil {
	return err
}

// Returns the decoded body, or an error for a failure response.
secret, err := httpclient.GetForObject[Secret](ctx, client, url,
	httpclient.BearerToken(token))

// Returns status code and either the success or the failure content.
entity, err := httpclient.PostForEntity[Receipt](ctx, client, url, secret)
if err == nil && entity.IsFailure() {
	failure, _ := entity.GetError()
	log.Printf("status %d: %s", entity.HTTPStatus(), failure.Message)
}

err = client.Put(ctx, url, secret)
err = client.Delete(ctx, url)
```

## Development

```shell
go build ./...
go test -race ./...                   # unit tests
go test -race -tags integration ./... # including tests that access the internet
```

## License

Carbyne Stack *HTTP Client* is open-sourced under the Apache License 2.0.
See the [LICENSE](LICENSE) file for details.

### 3rd Party Licenses

For information on how license obligations for 3rd party OSS dependencies are
fulfilled see the [README](https://github.com/carbynestack/carbynestack) file of
the Carbyne Stack repository.

## Contributing

Please see the Carbyne Stack
[Contributor's Guide](https://github.com/carbynestack/carbynestack/blob/master/CONTRIBUTING.md).
