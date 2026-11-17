// Package strombauer is a Go client for the Strombauer Shopware Store API.
//
// The client wraps the generated SDK in ./api and adds the two headers every
// request needs, keeps the session's sw-context-token up to date, and turns
// error responses into typed *APIError values.
package strombauer

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mheers/shop-strombauer-api-golang/api"
)

const (
	// DefaultBaseURL is the public shop that serves the Store API.
	DefaultBaseURL = "https://shop.strombauer.com"

	// Version is the version reported in the default User-Agent.
	Version = "0.1.0"

	// DefaultUserAgent is sent when no other User-Agent is configured.
	DefaultUserAgent = "shop-strombauer-api-golang/" + Version
)

// ErrAccessKeyRequired is returned by New when no sw-access-key is given.
var ErrAccessKeyRequired = errors.New("strombauer: access key is required")

// ErrUnexpectedResponse is returned when a response has a successful status
// code but a body that does not match the generated response type.
var ErrUnexpectedResponse = errors.New("strombauer: unexpected response")

// Client is a Strombauer Store API client. It is safe for concurrent use.
type Client struct {
	accessKey string
	baseURL   string
	userAgent string

	httpClient *http.Client
	api        *api.ClientWithResponses

	mu           sync.RWMutex
	contextToken string
}

// Option configures a Client.
type Option func(*Client) error

// New creates a client for the Strombauer Store API.
//
// accessKey is the sales channel's sw-access-key. It is required for every
// request and can be obtained from the shop operator.
func New(accessKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(accessKey) == "" {
		return nil, ErrAccessKeyRequired
	}

	client := &Client{
		accessKey: strings.TrimSpace(accessKey),
		baseURL:   DefaultBaseURL,
		userAgent: DefaultUserAgent,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	client.baseURL = strings.TrimRight(client.baseURL, "/")
	if client.baseURL == "" {
		return nil, errors.New("strombauer: base URL is empty")
	}

	doer := &authDoer{next: client.httpClient, client: client}
	apiClient, err := api.NewClientWithResponses(client.baseURL, api.WithHTTPClient(doer))
	if err != nil {
		return nil, fmt.Errorf("strombauer: create API client: %w", err)
	}
	client.api = apiClient

	return client, nil
}

// WithBaseURL points the client at another shop. The default is DefaultBaseURL.
func WithBaseURL(rawURL string) Option {
	return func(c *Client) error {
		parsed, err := url.Parse(strings.TrimSpace(rawURL))
		if err != nil {
			return fmt.Errorf("strombauer: invalid base URL: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("strombauer: base URL must use http or https, got %q", parsed.Scheme)
		}
		if parsed.Host == "" {
			return errors.New("strombauer: base URL has no host")
		}
		c.baseURL = rawURL
		return nil
	}
}

// WithHTTPClient replaces the default HTTP client (30 second timeout).
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		if httpClient == nil {
			return errors.New("strombauer: HTTP client is nil")
		}
		c.httpClient = httpClient
		return nil
	}
}

// WithContextToken sets the sw-context-token used for the first request.
// Later responses can replace it automatically.
func WithContextToken(token string) Option {
	return func(c *Client) error {
		c.contextToken = strings.TrimSpace(token)
		return nil
	}
}

// WithUserAgent replaces the default User-Agent.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) error {
		c.userAgent = strings.TrimSpace(userAgent)
		return nil
	}
}

// API returns the generated client. Use it for endpoints that have no typed
// convenience method.
func (c *Client) API() *api.ClientWithResponses {
	return c.api
}

// BaseURL returns the shop base URL the client talks to.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// ContextToken returns the current sw-context-token, or "" before a login.
func (c *Client) ContextToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.contextToken
}

// SetContextToken replaces the session token. It is called automatically when
// a response contains a new sw-context-token header.
func (c *Client) SetContextToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.contextToken = strings.TrimSpace(token)
}

// authDoer adds the access key and the current context token to every request
// and stores a new context token from every response.
type authDoer struct {
	next   api.HttpRequestDoer
	client *Client
}

func (d *authDoer) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("sw-access-key", d.client.accessKey)
	if token := d.client.ContextToken(); token != "" {
		req.Header.Set("sw-context-token", token)
	}
	if d.client.userAgent != "" {
		req.Header.Set("User-Agent", d.client.userAgent)
	}

	resp, err := d.next.Do(req)
	if err != nil {
		return nil, err
	}
	if token := resp.Header.Get("sw-context-token"); token != "" {
		d.client.SetContextToken(token)
	}
	return resp, nil
}
