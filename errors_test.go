package strombauer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name string
		err  *APIError
		want []string
	}{
		{
			name: "single error",
			err: &APIError{
				StatusCode: http.StatusUnauthorized,
				Errors: []APIErrorEntry{{
					Status: "401",
					Code:   "CHECKOUT__CUSTOMER_LOGIN_REQUIRED",
					Detail: "Bitte melde dich an.",
				}},
			},
			want: []string{"401", "CHECKOUT__CUSTOMER_LOGIN_REQUIRED", "Bitte melde dich an."},
		},
		{
			name: "several errors",
			err: &APIError{
				StatusCode: http.StatusBadRequest,
				Errors: []APIErrorEntry{
					{Status: "400", Code: "VIOLATION::IS_BLANK_ERROR", Detail: "Bitte gib die Straße an.", Source: &APIErrorSource{Pointer: "/street"}},
					{Status: "400", Code: "VIOLATION::IS_BLANK_ERROR", Detail: "Bitte gib die Stadt an.", Source: &APIErrorSource{Pointer: "/city"}},
				},
			},
			want: []string{"400", "VIOLATION::IS_BLANK_ERROR", "/street"},
		},
		{
			name: "unstructured body",
			err:  &APIError{StatusCode: http.StatusBadGateway, Body: "<html>bad gateway</html>"},
			want: []string{"502", "bad gateway"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, part := range tt.want {
				if !strings.Contains(msg, part) {
					t.Errorf("Error() = %q, want it to contain %q", msg, part)
				}
			}
		})
	}
}

func TestAPIErrorWithoutCodeHasNoDoubleSpace(t *testing.T) {
	err := &APIError{
		StatusCode: http.StatusBadRequest,
		Errors:     []APIErrorEntry{{Status: "400", Detail: "Bitte prüfe die Eingabe."}},
	}
	if got := err.Error(); strings.Contains(got, "  ") {
		t.Fatalf("Error() = %q, want no double space", got)
	}
}

func TestAPIErrorHasCode(t *testing.T) {
	err := &APIError{Errors: []APIErrorEntry{
		{Code: "VIOLATION::IS_BLANK_ERROR", Source: &APIErrorSource{Pointer: "/street"}},
	}}

	if !err.HasCode("VIOLATION::IS_BLANK_ERROR") {
		t.Error("HasCode returned false for a known code")
	}
	if err.HasCode("UNKNOWN") {
		t.Error("HasCode returned true for an unknown code")
	}
}

func TestIsCode(t *testing.T) {
	apiErr := &APIError{Errors: []APIErrorEntry{{Code: "CHECKOUT__CART_LOCKED"}}}
	wrapped := fmt.Errorf("place order: %w", apiErr)

	if !IsCode(wrapped, "CHECKOUT__CART_LOCKED") {
		t.Error("IsCode did not find a wrapped API error")
	}
	if IsCode(wrapped, "OTHER") {
		t.Error("IsCode found an unrelated code")
	}
	if IsCode(errors.New("plain error"), "CHECKOUT__CART_LOCKED") {
		t.Error("IsCode matched a non-API error")
	}
}

func TestResponseErrorParsesErrorList(t *testing.T) {
	body := []byte(`{"errors":[{"status":"400","code":"VIOLATION::IS_BLANK_ERROR","detail":"Bitte gib die Telefonnummer an.","source":{"pointer":"/phoneNumber"}}]}`)

	err := responseError(http.StatusBadRequest, body)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d", apiErr.StatusCode)
	}
	if len(apiErr.Errors) != 1 {
		t.Fatalf("len(Errors) = %d", len(apiErr.Errors))
	}
	if apiErr.Errors[0].Source == nil || apiErr.Errors[0].Source.Pointer != "/phoneNumber" {
		t.Errorf("source pointer = %#v", apiErr.Errors[0].Source)
	}
}

func TestResponseErrorAcceptsSuccess(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusNoContent} {
		if err := responseError(status, nil); err != nil {
			t.Errorf("status %d: unexpected error %v", status, err)
		}
	}
}

func TestResponseErrorKeepsUnstructuredBody(t *testing.T) {
	err := responseError(http.StatusInternalServerError, []byte("upstream exploded"))
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Body != "upstream exploded" {
		t.Errorf("Body = %q", apiErr.Body)
	}
}

func TestUnexpectedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	client, err := New("access-key", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.Customer(context.Background())
	if err == nil {
		t.Fatal("expected an error for an invalid JSON response")
	}
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("error = %v, want ErrUnexpectedResponse", err)
	}
}
