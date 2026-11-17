package strombauer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mheers/shop-strombauer-api-golang/api"
)

func TestProductsPassesQueryParameters(t *testing.T) {
	var gotPath, gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"apiAlias":"product_list","elements":[{"id":"p1","productNumber":"1001","name":"Battery","apiAlias":"product","bundleProducts":[],"properties":[],"tags":[],"variants":[],"productType":"PRODUCT"}],"limit":10,"page":2,"sort":"-popularity","total":1}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	page := api.CriteriaPage(2)
	limit := api.CriteriaLimit(10)
	sort := api.GetStoreApiProductSearchParamsSort("-popularity")

	resp, err := client.Products(context.Background(), &api.GetStoreApiProductSearchParams{
		Page:  &page,
		Limit: &limit,
		Sort:  &sort,
	})
	if err != nil {
		t.Fatalf("Products: %v", err)
	}

	if gotPath != "/store-api/v1/product" {
		t.Errorf("path = %q", gotPath)
	}
	for _, part := range []string{"page=2", "limit=10", "sort=-popularity"} {
		if !containsQuery(gotQuery, part) {
			t.Errorf("query %q does not contain %q", gotQuery, part)
		}
	}
	if len(resp.Elements) != 1 || resp.Elements[0].Id != "p1" {
		t.Fatalf("elements = %#v", resp.Elements)
	}
}

func TestProductUsesProductPath(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"p1","apiAlias":"product_detail","bundleProducts":[],"crossSellings":[],"downloads":[],"images":[],"properties":[],"tags":[],"variants":[]}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	product, err := client.Product(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Product: %v", err)
	}
	if gotPath != "/store-api/v1/product/p1" {
		t.Errorf("path = %q", gotPath)
	}
	if product.Id != "p1" {
		t.Errorf("id = %q", product.Id)
	}
}

func TestSearchPassesTermAndSource(t *testing.T) {
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"apiAlias":"product_search","elements":[],"limit":10,"page":1,"searchLogId":null,"sort":"relevance","total":0}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	term := api.CriteriaTerm("battery")
	source := api.GetStoreApiSearchParamsSource("results")

	if _, err := client.Search(context.Background(), &api.GetStoreApiSearchParams{Term: &term, Source: &source}); err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, part := range []string{"term=battery", "source=results"} {
		if !containsQuery(gotQuery, part) {
			t.Errorf("query %q does not contain %q", gotQuery, part)
		}
	}
}

func TestCountriesDecodesList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"items":[{"id":"c1","iso":"DE","name":"Germany","default":true,"postalCodeRequired":true}]}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	countries, err := client.Countries(context.Background())
	if err != nil {
		t.Fatalf("Countries: %v", err)
	}
	if len(countries.Items) != 1 || countries.Items[0].Iso != "DE" {
		t.Fatalf("items = %#v", countries.Items)
	}
}

func TestManufacturers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/store-api/v1/manufacturers" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"elements":[{"id":"m1","name":"Huawei","productCount":2}],"limit":10,"page":1,"total":1}`)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	manufacturers, err := client.Manufacturers(context.Background(), nil)
	if err != nil {
		t.Fatalf("Manufacturers: %v", err)
	}
	if len(manufacturers.Elements) != 1 || manufacturers.Elements[0].Id != "m1" {
		t.Fatalf("elements = %#v", manufacturers.Elements)
	}
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := New("access-key", WithBaseURL(baseURL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func containsQuery(query, part string) bool {
	for _, pair := range splitAnd(query) {
		if pair == part {
			return true
		}
	}
	return false
}

func splitAnd(query string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '&' {
			parts = append(parts, query[start:i])
			start = i + 1
		}
	}
	if start <= len(query) {
		parts = append(parts, query[start:])
	}
	return parts
}
