//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mheers/shop-strombauer-api-golang"
	"github.com/mheers/shop-strombauer-api-golang/api"
)

const requestTimeout = 60 * time.Second

func newClient(t *testing.T) *strombauer.Client {
	t.Helper()

	accessKey := os.Getenv("STROMBAUER_ACCESS_KEY")
	if accessKey == "" {
		t.Skip("STROMBAUER_ACCESS_KEY is not set; copy .env.example to .env and fill it in")
	}

	options := []strombauer.Option{
		strombauer.WithHTTPClient(&http.Client{Timeout: requestTimeout}),
	}
	if baseURL := os.Getenv("STROMBAUER_BASE_URL"); baseURL != "" {
		options = append(options, strombauer.WithBaseURL(baseURL))
	}
	if token := os.Getenv("STROMBAUER_CONTEXT_TOKEN"); token != "" {
		options = append(options, strombauer.WithContextToken(token))
	}

	client, err := strombauer.New(accessKey, options...)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	return client
}

func testContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 90*time.Second)
}

// TestReadOnlyPublicEndpoints exercises endpoints that do not need a customer
// session. Every call is a GET.
func TestReadOnlyPublicEndpoints(t *testing.T) {
	client := newClient(t)
	ctx, cancel := testContext(t)
	defer cancel()

	countries, err := client.Countries(ctx)
	if err != nil {
		t.Fatalf("countries: %v", err)
	}
	if len(countries.Items) == 0 {
		t.Fatal("countries: empty list")
	}
	t.Logf("countries: %d", len(countries.Items))

	limit := api.CriteriaLimit(1)
	products, err := client.Products(ctx, &api.GetStoreApiProductSearchParams{Limit: &limit})
	if err != nil {
		t.Fatalf("products: %v", err)
	}
	t.Logf("products: total=%d returned=%d", products.Total, len(products.Elements))

	categories, err := client.Categories(ctx, &api.GetStoreApiCategorySearchParams{Limit: &limit})
	if err != nil {
		t.Fatalf("categories: %v", err)
	}
	if categories.Elements == nil {
		t.Log("categories: no elements")
	} else {
		t.Logf("categories: %d", len(*categories.Elements))
	}

	manufacturers, err := client.Manufacturers(ctx, nil)
	if err != nil {
		t.Fatalf("manufacturers: %v", err)
	}
	t.Logf("manufacturers: %d", len(manufacturers.Elements))

	// A search without "source" is not recorded by the shop, so this stays
	// read-only.
	term := api.CriteriaTerm("battery")
	search, err := client.Search(ctx, &api.GetStoreApiSearchParams{Term: &term})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	searchLogID := ""
	if search.SearchLogId != nil {
		searchLogID = *search.SearchLogId
	}
	t.Logf("search: total=%d searchLogId=%q", search.Total, searchLogID)
}

// TestReadOnlyAuthenticatedEndpoints exercises the customer and sales-order
// reads. It uses STROMBAUER_CONTEXT_TOKEN when set and never logs out, which
// keeps the run strictly read-only. Otherwise it logs in with
// STROMBAUER_USERNAME/STROMBAUER_PASSWORD and logs out again.
func TestReadOnlyAuthenticatedEndpoints(t *testing.T) {
	client := newClient(t)
	ctx, cancel := testContext(t)
	defer cancel()

	if client.ContextToken() == "" {
		username := os.Getenv("STROMBAUER_USERNAME")
		password := os.Getenv("STROMBAUER_PASSWORD")
		if username == "" || password == "" {
			t.Skip("set STROMBAUER_CONTEXT_TOKEN, or STROMBAUER_USERNAME and STROMBAUER_PASSWORD")
		}

		if err := client.Login(ctx, username, password); err != nil {
			t.Fatalf("login: %v", err)
		}
		defer func() {
			if err := client.Logout(ctx); err != nil {
				t.Errorf("logout: %v", err)
			}
		}()
	}

	customer, err := client.Customer(ctx)
	if err != nil {
		t.Fatalf("customer: %v", err)
	}
	if customer.Customer.Email == "" {
		t.Error("customer: empty e-mail address")
	}
	t.Logf("customer: %s %s, state=%s", customer.Customer.FirstName, customer.Customer.LastName, customer.CustomerState)

	limit := 1
	orders, err := client.Orders(ctx, &api.GetStoreApiStrombauerSalesOrdersParams{Limit: &limit})
	if err != nil {
		t.Fatalf("orders: %v", err)
	}
	t.Logf("orders: returned=%d pending=%d", len(orders.Items), orders.PendingOrderCount)
}
