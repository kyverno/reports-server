//go:build fips
// +build fips

package main

/*
This file will only be compiled when BUILD_TAGS=fips.


Package fipsonly enforces FIPS settings via init() function which
  1. Forces the application to use FIPS-compliant TLS configurations.
  2. Restricts cryptographic operations to those allowed under FIPS standards.

This package is available when Go is compiled with GOEXPERIMENT=systemcrypto for Go version 1.21 and above.

Refer Link: https://go.dev/src/crypto/tls/fipsonly/fipsonly.go
*/

import (
	_ "crypto/tls/fipsonly"
) //nolint:golint,unused
