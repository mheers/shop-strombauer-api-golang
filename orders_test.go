package strombauer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mheers/shop-strombauer-api-golang/api"
)

func TestOrdersPassesFilters(t *testing.T) {
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"filterCounts":{"OPEN_INVOICES":0,"OPEN_DELIVERIES":0,"ACTION_REQUIRED":0},"items":[],"syncState":"DONE"}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	term := "AUF26"
	filter := []api.GetStoreApiStrombauerSalesOrdersParamsFilter{"OPEN_INVOICES"}
	page := 2
	limit := 50

	if _, err := client.Orders(context.Background(), &api.GetStoreApiStrombauerSalesOrdersParams{
		Term:   &term,
		Filter: &filter,
		P:      &page,
		Limit:  &limit,
	}); err != nil {
		t.Fatalf("Orders: %v", err)
	}

	for _, part := range []string{"term=AUF26", "filter=OPEN_INVOICES", "p=2", "limit=50"} {
		if !containsQuery(gotQuery, part) {
			t.Errorf("query %q does not contain %q", gotQuery, part)
		}
	}
}

func TestOrderUsesOrderPath(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.Order(context.Background(), "order-1"); err != nil {
		t.Fatalf("Order: %v", err)
	}
	if gotPath != "/store-api/v1/sales-order/order-1" {
		t.Errorf("path = %q", gotPath)
	}
}
