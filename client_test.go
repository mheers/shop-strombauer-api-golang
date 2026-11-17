package strombauer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestNewRequiresAccessKey(t *testing.T) {
	if _, err := New(""); !errors.Is(err, ErrAccessKeyRequired) {
		t.Fatalf("New(\"\") error = %v, want %v", err, ErrAccessKeyRequired)
	}
	if _, err := New("   "); !errors.Is(err, ErrAccessKeyRequired) {
		t.Fatalf("New(\"   \") error = %v, want %v", err, ErrAccessKeyRequired)
	}
}

func TestNewRejectsInvalidBaseURL(t *testing.T) {
	if _, err := New("key", WithBaseURL("://nope")); err == nil {
		t.Fatal("expected an error for an invalid base URL")
	}
}

func TestClientSendsAuthenticationHeaders(t *testing.T) {
	var gotAccessKey, gotContextToken, gotUserAgent, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccessKey = r.Header.Get("sw-access-key")
		gotContextToken = r.Header.Get("sw-context-token")
		gotUserAgent = r.Header.Get("User-Agent")
		gotPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"addresses":[],"customer":{},"metrics":{},"addressFields":{}}`)
	}))
	defer srv.Close()

	client, err := New("access-key",
		WithBaseURL(srv.URL),
		WithContextToken("context-token"),
		WithUserAgent("strombauer-test/1.0"),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := client.Customer(context.Background()); err != nil {
		t.Fatalf("Customer: %v", err)
	}

	if gotPath != "/store-api/v1/account/customer" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAccessKey != "access-key" {
		t.Errorf("sw-access-key = %q", gotAccessKey)
	}
	if gotContextToken != "context-token" {
		t.Errorf("sw-context-token = %q", gotContextToken)
	}
	if gotUserAgent != "strombauer-test/1.0" {
		t.Errorf("User-Agent = %q", gotUserAgent)
	}
}

func TestClientCapturesContextTokenFromLogin(t *testing.T) {
	var loginBody map[string]string
	var customerToken string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/store-api/v1/account/login":
			if err := json.NewDecoder(r.Body).Decode(&loginBody); err != nil {
				t.Errorf("decode login body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("sw-context-token", "new-session-token")
			_, _ = io.WriteString(w, `{}`)
		case "/store-api/v1/account/customer":
			customerToken = r.Header.Get("sw-context-token")
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"addresses":[],"customer":{},"metrics":{},"addressFields":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := New("access-key", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client.ContextToken() != "" {
		t.Fatalf("initial token = %q, want empty", client.ContextToken())
	}

	if err := client.Login(context.Background(), "max@example.com", "secret"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if got := client.ContextToken(); got != "new-session-token" {
		t.Fatalf("ContextToken = %q, want new-session-token", got)
	}
	if loginBody["username"] != "max@example.com" || loginBody["password"] != "secret" {
		t.Fatalf("login body = %#v", loginBody)
	}

	if _, err := client.Customer(context.Background()); err != nil {
		t.Fatalf("Customer: %v", err)
	}
	if customerToken != "new-session-token" {
		t.Fatalf("customer request token = %q, want captured token", customerToken)
	}
}

func TestClientContextTokenIsSafeForConcurrentUse(t *testing.T) {
	client, err := New("access-key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.SetContextToken("token")
			_ = client.ContextToken()
		}()
	}
	wg.Wait()

	if got := client.ContextToken(); got != "token" {
		t.Fatalf("ContextToken = %q, want token", got)
	}
}

func TestLoginReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"errors":[{"status":"401","code":"CHECKOUT__CUSTOMER_AUTH_BAD_CREDENTIALS","detail":"E-Mail-Adresse oder Passwort ist falsch."}]}`)
	}))
	defer srv.Close()

	client, err := New("access-key", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = client.Login(context.Background(), "max@example.com", "wrong")
	if err == nil {
		t.Fatal("expected a login error")
	}
	if !IsCode(err, "CHECKOUT__CUSTOMER_AUTH_BAD_CREDENTIALS") {
		t.Fatalf("error %v does not carry the expected code", err)
	}
	if !strings.Contains(err.Error(), "E-Mail-Adresse oder Passwort ist falsch.") {
		t.Fatalf("error %q does not contain the server detail", err)
	}
}
