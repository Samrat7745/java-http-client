// Copyright (c) 2021-2026 - for information on the respective copyright owner
// see the NOTICE file and/or the repository https://github.com/carbynestack/java-http-client.
//
// SPDX-License-Identifier: Apache-2.0

package httpclient

// Header is a single HTTP request header.
type Header struct {
	Name  string
	Value string
}

const bearerPrefix = "Bearer"

// BearerToken returns the Authorization header carrying the given bearer token.
func BearerToken(token string) Header {
	return Header{Name: "Authorization", Value: bearerPrefix + " " + token}
}
