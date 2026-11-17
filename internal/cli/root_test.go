package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mheers/shop-strombauer-api-golang"
)

func TestProductsListJSON(t *testing.T) {
	var gotAccessKey, gotPath, gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccessKey = r.Header.Get("sw-access-key")
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"apiAlias":"dal_entity_search_result","elements":[{"id":"p1","productNumber":"1001","name":"Battery","apiAlias":"product","bundleProducts":[],"properties":[],"tags":[],"variants":[],"productType":"PRODUCT"}],"limit":10,"page":1,"sort":"-popularity","total":1}`)
	}))
	defer server.Close()

	configPath := writeConfig(t, &Config{BaseURL: server.URL, AccessKey: "test-key"})

	stdout, _, err := execute(t, Options{ConfigPath: configPath}, "products", "list", "--json", "--page", "2", "--limit", "10")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if gotAccessKey != "test-key" {
		t.Errorf("sw-access-key = %q", gotAccessKey)
	}
	if gotPath != "/store-api/v1/product" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotQuery, "page=2") || !strings.Contains(gotQuery, "limit=10") {
		t.Errorf("query = %q", gotQuery)
	}

	var response struct {
		Elements []struct {
			Id string `json:"id"`
		} `json:"elements"`
	}
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout)
	}
	if len(response.Elements) != 1 || response.Elements[0].Id != "p1" {
		t.Fatalf("elements = %#v", response.Elements)
	}
}

func TestLoginStoresContextToken(t *testing.T) {
	var loginBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&loginBody); err != nil {
			t.Errorf("decode login body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("sw-context-token", "session-token")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	configPath := writeConfig(t, &Config{BaseURL: server.URL, AccessKey: "test-key"})

	stdout, _, err := execute(t, Options{ConfigPath: configPath},
		"login", "--username", "max@example.com", "--password", "secret")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(stdout, "Logged in as max@example.com") {
		t.Errorf("stdout = %q", stdout)
	}
	if loginBody["username"] != "max@example.com" || loginBody["password"] != "secret" {
		t.Errorf("login body = %#v", loginBody)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if config.ContextToken != "session-token" {
		t.Errorf("stored context token = %q", config.ContextToken)
	}
}

func TestLoginReadsCredentialsFromStdin(t *testing.T) {
	var loginBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&loginBody); err != nil {
			t.Errorf("decode login body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("sw-context-token", "session-token")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	configPath := writeConfig(t, &Config{BaseURL: server.URL, AccessKey: "test-key"})

	stdout, stderr, err := execute(t, Options{
		ConfigPath: configPath,
		In:         strings.NewReader("max@example.com\nsecret\n"),
	}, "login")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if loginBody["username"] != "max@example.com" || loginBody["password"] != "secret" {
		t.Fatalf("login body = %#v", loginBody)
	}
	if strings.Contains(stdout, "E-mail:") || strings.Contains(stdout, "Password:") {
		t.Errorf("prompts leaked to stdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "E-mail:") || !strings.Contains(stderr, "Password:") {
		t.Errorf("prompts missing from stderr:\n%s", stderr)
	}
}

func TestSuccessfulCommandPersistsRotatedToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("sw-context-token", "rotated-token")
		_, _ = io.WriteString(w, `{"addresses":[],"customer":{"email":"max@example.com"},"metrics":{},"addressFields":{}}`)
	}))
	defer server.Close()

	configPath := writeConfig(t, &Config{
		BaseURL:      server.URL,
		AccessKey:    "test-key",
		ContextToken: "old-token",
	})

	if _, _, err := execute(t, Options{ConfigPath: configPath}, "customer", "--json"); err != nil {
		t.Fatalf("execute: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if config.ContextToken != "rotated-token" {
		t.Fatalf("stored token = %q, want rotated-token", config.ContextToken)
	}
}

func TestCustomerReturnsTypedAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"errors":[{"status":"401","code":"CHECKOUT__CUSTOMER_LOGIN_REQUIRED","detail":"Bitte melde dich an."}]}`)
	}))
	defer server.Close()

	configPath := writeConfig(t, &Config{BaseURL: server.URL, AccessKey: "test-key"})

	_, _, err := execute(t, Options{ConfigPath: configPath}, "customer")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strombauer.IsCode(err, "CHECKOUT__CUSTOMER_LOGIN_REQUIRED") {
		t.Fatalf("error = %v", err)
	}
}

func TestConfigSetAccessKeyAndShow(t *testing.T) {
	configPath := t.TempDir() + "/config.json"

	if _, _, err := execute(t, Options{ConfigPath: configPath}, "config", "set-access-key", "super-secret-key"); err != nil {
		t.Fatalf("set-access-key: %v", err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if config.AccessKey != "super-secret-key" {
		t.Fatalf("access key = %q", config.AccessKey)
	}

	stdout, _, err := execute(t, Options{ConfigPath: configPath}, "config", "show")
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if strings.Contains(stdout, "super-secret-key") {
		t.Fatalf("config show leaked the access key:\n%s", stdout)
	}
	if !strings.Contains(stdout, "********-key") {
		t.Fatalf("config show does not mask the key:\n%s", stdout)
	}
}

func TestCartAddUsesConfiguredBaseURL(t *testing.T) {
	var body struct {
		Items []struct {
			ProductId string `json:"productId"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/store-api/v1/checkout/cart/line-item" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"items":[],"selected":{},"available":{},"totals":{"net":0,"vat":[],"gross":0,"savings":null,"taxExemption":null},"notices":[]}`)
	}))
	defer server.Close()

	configPath := writeConfig(t, &Config{BaseURL: server.URL, AccessKey: "test-key"})

	if _, _, err := execute(t, Options{ConfigPath: configPath}, "cart", "add", "p1", "--quantity", "2"); err != nil {
		t.Fatalf("cart add: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].ProductId != "p1" || body.Items[0].Quantity != 2 {
		t.Fatalf("body = %#v", body)
	}
}

func TestCartShowPrintsLineItemIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"items":[{"id":"line-1","productId":"p1","productNumber":"1001","name":"Battery","quantity":2,"price":{"unit":10,"total":20},"listPrice":null,"measureUnit":{"code":"STÜCK","name":"Stk.","quantityPerUnit":1},"availableStock":5,"stockForecast":null}],"selected":{},"available":{},"totals":{"net":20,"vat":[],"gross":20,"savings":null,"taxExemption":null},"notices":[]}`)
	}))
	defer server.Close()

	configPath := writeConfig(t, &Config{BaseURL: server.URL, AccessKey: "test-key"})

	stdout, _, err := execute(t, Options{ConfigPath: configPath}, "cart", "show")
	if err != nil {
		t.Fatalf("cart show: %v", err)
	}
	for _, part := range []string{"line-1", "1001", "Battery", "2 Stk.", "20.00"} {
		if !strings.Contains(stdout, part) {
			t.Errorf("cart output does not contain %q:\n%s", part, stdout)
		}
	}
}

func execute(t *testing.T, opts Options, args ...string) (string, string, error) {
	t.Helper()

	var stdout, stderr strings.Builder
	opts.Out = &stdout
	opts.Err = &stderr
	if opts.In == nil {
		opts.In = strings.NewReader("")
	}
	if opts.Getenv == nil {
		opts.Getenv = func(string) string { return "" }
	}

	cmd := NewRootCmd(opts)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func writeConfig(t *testing.T, config *Config) string {
	t.Helper()

	path := t.TempDir() + "/config.json"
	if err := SaveConfig(path, config); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	return path
}
