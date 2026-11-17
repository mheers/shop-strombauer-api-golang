package strombauer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mheers/shop-strombauer-api-golang/api"
)

const emptyCart = `{"items":[],"selected":{},"available":{},"totals":{"net":0,"vat":[],"gross":0,"savings":null,"taxExemption":null},"notices":[]}`

func TestCartReadsCurrentCart(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, emptyCart)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	cart, err := client.Cart(context.Background())
	if err != nil {
		t.Fatalf("Cart: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/store-api/v1/checkout/cart" {
		t.Errorf("request = %s %s", gotMethod, gotPath)
	}
	if cart == nil {
		t.Fatal("cart is nil")
	}
}

func TestAddProductSendsCartItems(t *testing.T) {
	var body api.CartItems

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/store-api/v1/checkout/cart/line-item" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, emptyCart)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.AddProduct(context.Background(), "p1", 3); err != nil {
		t.Fatalf("AddProduct: %v", err)
	}

	if len(body.Items) != 1 {
		t.Fatalf("items = %#v", body.Items)
	}
	if body.Items[0].ProductId != "p1" || body.Items[0].Quantity == nil || *body.Items[0].Quantity != 3 {
		t.Fatalf("item = %#v", body.Items[0])
	}
}

func TestAddProductDefaultsQuantityToOne(t *testing.T) {
	var body api.CartItems

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, emptyCart)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.AddProduct(context.Background(), "p1", 0); err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if body.Items[0].Quantity == nil || *body.Items[0].Quantity != 1 {
		t.Fatalf("quantity = %#v, want 1", body.Items[0].Quantity)
	}
}

func TestUpdateCartItem(t *testing.T) {
	var gotPath string
	var body api.PatchStoreApiStrombauerCheckoutCartUpdateItemJSONBody

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, emptyCart)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.UpdateCartItem(context.Background(), "line-1", 4); err != nil {
		t.Fatalf("UpdateCartItem: %v", err)
	}
	if gotPath != "/store-api/v1/checkout/cart/line-item/line-1" {
		t.Errorf("path = %q", gotPath)
	}
	if body.Quantity != 4 {
		t.Errorf("quantity = %d", body.Quantity)
	}
}

func TestRemoveCartItem(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, emptyCart)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.RemoveCartItem(context.Background(), "line-1"); err != nil {
		t.Fatalf("RemoveCartItem: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/store-api/v1/checkout/cart/line-item/line-1" {
		t.Errorf("request = %s %s", gotMethod, gotPath)
	}
}

func TestClearCart(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.ClearCart(context.Background()); err != nil {
		t.Fatalf("ClearCart: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/store-api/v1/checkout/cart" {
		t.Errorf("request = %s %s", gotMethod, gotPath)
	}
}

func TestSetPaymentMethod(t *testing.T) {
	var body api.PostStoreApiStrombauerCheckoutSetPaymentMethodJSONBody

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/store-api/v1/checkout/set-payment-method" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, emptyCart)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.SetPaymentMethod(context.Background(), "pm-1"); err != nil {
		t.Fatalf("SetPaymentMethod: %v", err)
	}
	if body.PaymentMethodId != "pm-1" {
		t.Errorf("paymentMethodId = %q", body.PaymentMethodId)
	}
}

func TestSetShippingMethod(t *testing.T) {
	var body api.PostStoreApiStrombauerCheckoutSetShippingMethodJSONBody

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, emptyCart)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.SetShippingMethod(context.Background(), "sm-1"); err != nil {
		t.Fatalf("SetShippingMethod: %v", err)
	}
	if body.ShippingMethodId != "sm-1" {
		t.Errorf("shippingMethodId = %q", body.ShippingMethodId)
	}
}

func TestPlaceOrderReturnsOrderNumber(t *testing.T) {
	var body api.PostStoreApiCheckoutCartOrderJSONBody

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/store-api/v1/checkout/order" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"number":"AUF26-9217"}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	number, err := client.PlaceOrder(context.Background(), true)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if number != "AUF26-9217" {
		t.Errorf("number = %q", number)
	}
	if !bool(body.TermsAccepted) {
		t.Error("termsAccepted was not sent")
	}
}

func TestPlaceOrderRequiresTerms(t *testing.T) {
	client := newTestClient(t, "http://127.0.0.1:1")
	_, err := client.PlaceOrder(context.Background(), false)
	if !errors.Is(err, ErrTermsNotAccepted) {
		t.Fatalf("error = %v, want ErrTermsNotAccepted", err)
	}
}
