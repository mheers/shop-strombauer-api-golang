// Package e2e contains read-only end-to-end tests against the live shop.
//
// The tests are behind the "e2e" build tag so that a normal `go test ./...`
// never talks to the network:
//
//	go test -tags=e2e ./e2e
//
// Credentials are read from the environment. See .env.example and the README.
package e2e
