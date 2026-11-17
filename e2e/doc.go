// Package e2e contains read-only end-to-end tests against the live shop.
//
// The tests are behind the "e2e" build tag so that a normal `go test ./...`
// never talks to the network:
//
//	go test -tags=e2e ./e2e
//
// The tests only read. When STROMBAUER_CONTEXT_TOKEN is not set and
// STROMBAUER_USERNAME/STROMBAUER_PASSWORD are, the authenticated test logs in
// and out again; those two calls only create and destroy a session.
//
// Credentials are read from the environment. See .env.example and the README.
package e2e
