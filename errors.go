package strombauer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// APIError is returned for every non-2xx response of the Store API. The shop
// returns a list of machine-readable errors; decide by Code, never by Detail.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int

	// Errors is the parsed error list. It is empty when the server returned a
	// body that is not a Store API error document.
	Errors []APIErrorEntry

	// Body is the raw response body, kept for unstructured errors.
	Body string
}

// APIErrorEntry is one entry of the Store API error list.
type APIErrorEntry struct {
	// Status is the HTTP status as a string.
	Status string `json:"status"`

	// Code is the stable, machine-readable error code.
	Code string `json:"code"`

	// Detail is the German sentence for the customer.
	Detail string `json:"detail"`

	// Source names the field the error belongs to, if any.
	Source *APIErrorSource `json:"source,omitempty"`

	// Meta carries additional numbers for a few error codes.
	Meta map[string]any `json:"meta,omitempty"`
}

// APIErrorSource points at the request field an error belongs to.
type APIErrorSource struct {
	Pointer string `json:"pointer"`
}

// Error implements error.
func (e *APIError) Error() string {
	if len(e.Errors) == 0 {
		body := strings.TrimSpace(e.Body)
		if body == "" {
			return fmt.Sprintf("strombauer: request failed with status %d", e.StatusCode)
		}
		return fmt.Sprintf("strombauer: request failed with status %d: %s", e.StatusCode, truncate(body, 300))
	}

	first := e.Errors[0]
	status := first.Status
	if status == "" {
		status = fmt.Sprintf("%d", e.StatusCode)
	}

	message := fmt.Sprintf("strombauer: %s %s", status, first.Code)
	if first.Detail != "" {
		message += ": " + first.Detail
	}
	if first.Source != nil && first.Source.Pointer != "" {
		message += " (" + first.Source.Pointer + ")"
	}
	if len(e.Errors) > 1 {
		message += fmt.Sprintf(" (and %d more errors)", len(e.Errors)-1)
	}
	return message
}

// HasCode reports whether the response contains an error with the given code.
func (e *APIError) HasCode(code string) bool {
	for _, entry := range e.Errors {
		if entry.Code == code {
			return true
		}
	}
	return false
}

// IsCode reports whether err is or wraps an *APIError with the given code.
func IsCode(err error, code string) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.HasCode(code)
}

// AsAPIError extracts an *APIError from err.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return nil, false
	}
	return apiErr, true
}

// responseError converts a non-2xx response into an *APIError.
func responseError(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	apiErr := &APIError{StatusCode: statusCode}

	var payload struct {
		Errors []APIErrorEntry `json:"errors"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		apiErr.Errors = payload.Errors
	}
	if len(apiErr.Errors) == 0 {
		apiErr.Body = string(body)
	}
	return apiErr
}

// wrapRequestError marks response decoding errors as unexpected responses while
// leaving transport errors untouched.
func wrapRequestError(err error) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
		return fmt.Errorf("%w: %v", ErrUnexpectedResponse, err)
	}
	return err
}

// unexpectedResponse reports a successful status code with an unusable body.
func unexpectedResponse(statusCode int, body []byte) error {
	return fmt.Errorf("%w (status %d): %s", ErrUnexpectedResponse, statusCode, truncate(string(body), 300))
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max]) + "…"
}
