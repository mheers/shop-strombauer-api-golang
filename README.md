# shop-strombauer-api-golang

A Go client library and command line client for the **Strombauer Shopware Store
API**.

The library is derived from the official Strombauer API documentation and
OpenAPI specification:

- Documentation: <https://shop.strombauer.com/store-api/v1/docs>
- OpenAPI 3.0 document: `spec/openapi.json` (title
  *Strombauer Shopware API*, version `1.0.0`, 79 paths, 323 schemas)

The API description, header rules, flows and error codes below are taken from
that documentation. The vendored specification is the source of truth for the
generated Go types and client.

## Contents

- [Features](#features)
- [Install](#install)
- [Library quick start](#library-quick-start)
- [Authentication and sessions](#authentication-and-sessions)
- [What the API covers](#what-the-api-covers)
- [Error handling](#error-handling)
- [CLI](#cli)
- [Generated SDK](#generated-sdk)
- [Project layout](#project-layout)
- [Development](#development)
- [License](#license)

## Features

- Typed Go SDK generated from the OpenAPI 3.0 document with
  `oapi-codegen` v2 (`api/api.gen.go`).
- Hand-written wrapper in the module root that adds the two headers every
  request needs, captures new session tokens automatically, and returns typed
  errors.
- Convenience methods for the common flows: login, customer, catalog, cart,
  checkout and sales orders.
- The complete generated client is always available through `client.API()` for
  endpoints without a convenience method.
- Cobra CLI with table output, `--json` output, a `0600` config file and
  environment-variable support.
- Unit tests use `httptest`; no live shop account is required.

## Install

Library:

```sh
go get github.com/mheers/shop-strombauer-api-golang
```

CLI:

```sh
go install github.com/mheers/shop-strombauer-api-golang/cmd/strombauer@latest
```

Go 1.26 or newer is required.

## Library quick start

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mheers/shop-strombauer-api-golang"
	"github.com/mheers/shop-strombauer-api-golang/api"
)

func main() {
	ctx := context.Background()

	client, err := strombauer.New(os.Getenv("STROMBAUER_ACCESS_KEY"))
	if err != nil {
		panic(err)
	}

	if err := client.Login(ctx, "max@example.com", os.Getenv("STROMBAUER_PASSWORD")); err != nil {
		panic(err)
	}
	defer func() { _ = client.Logout(ctx) }()

	customer, err := client.Customer(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println(customer.Customer.FirstName, customer.Customer.LastName, customer.CustomerState)

	page := api.CriteriaPage(1)
	limit := api.CriteriaLimit(10)
	products, err := client.Products(ctx, &api.GetStoreApiProductSearchParams{
		Page:  &page,
		Limit: &limit,
	})
	if err != nil {
		panic(err)
	}

	cart, err := client.AddProduct(ctx, products.Elements[0].Id, 1)
	if err != nil {
		panic(err)
	}
	if cart.Totals.Gross != nil {
		fmt.Printf("cart total: %.2f EUR\n", *cart.Totals.Gross)
	}
}
```

Persist the session token between processes:

```go
token := client.ContextToken()

// later, in another process
client, err := strombauer.New(accessKey, strombauer.WithContextToken(token))
```

The client is safe for concurrent use. `WithContextToken` only sets the initial
value; every response that contains a new `sw-context-token` header replaces it
automatically.

## Authentication and sessions

The Store API uses two headers:

- `sw-access-key` is required for **every** request. It identifies the sales
  channel and is provided by the shop operator.
- `sw-context-token` identifies the session. Endpoints that require a logged-in
  customer need it in addition to the access key. Log in with
  `POST /store-api/v1/account/login`; the response header contains the token.
  Whenever a response contains a new `sw-context-token`, use that token from
  then on. The Go client does this automatically.

Endpoints that need a logged-in customer answer `401` with
`CHECKOUT__CUSTOMER_LOGIN_REQUIRED` when no session exists, including for
guests. A wrong e-mail address or password at login stays `401` with
`CHECKOUT__CUSTOMER_AUTH_BAD_CREDENTIALS`.

The API also supports passkey login and registration. The generated client
exposes those endpoints through `client.API()`; the convenience wrapper covers
the password login.

## What the API covers

All paths start with `/store-api/v1/`. The base URL is
`https://shop.strombauer.com`. The documented endpoint groups are:

| Area | Examples |
|---|---|
| Registration | `account/email-check`, `account/business-register`, onboarding booking slots |
| Login | login, logout, password recovery, account activation |
| Passkeys | login/register options, login, register, list, delete |
| Customer | customer data, addresses, password change |
| Shipping addresses | create, update, delete, set default |
| Countries | country list for registration and delivery |
| Catalog | products, product detail, search, categories, manufacturers, tags |
| Cart | read, add line item, update, remove, clear |
| Stored carts | list, create, update, delete |
| Checkout | shipping/payment/address/date/comment/reference, place order, quote request |
| Sales orders | list, detail, statuses, document download |
| Price list export | start export, poll job, download result |

### Registration flow

1. `POST /store-api/v1/account/email-check` — with `mode: login` the customer
   exists and continues with the login; with `mode: register` continue with
   step 2.
2. `POST /store-api/v1/account/business-register` — creates the company
   account. The billing country comes from `GET /store-api/v1/country`. The
   response header contains the token of the now logged-in customer.
3. `GET /store-api/v1/onboarding/booking-slots` — free slots for the
   onboarding call.
4. `POST /store-api/v1/onboarding/booking` — books the chosen slot.

Prices and ordering are unlocked after the onboarding call.

### Order flow

1. Log in: `POST /store-api/v1/account/login`.
2. Find products: `GET /store-api/v1/product` or `GET /store-api/v1/search`.
3. Fill the cart: `POST /store-api/v1/checkout/cart/line-item`.
4. Choose shipping and payment with the checkout endpoints. The allowed values
   are listed in the cart under `available`.
5. Place the order: `POST /store-api/v1/checkout/order`.
6. Follow the order in `GET /store-api/v1/sales-orders`. Until it appears there,
   usually within a few minutes, `pendingOrderCount` counts it.

Instead of steps 4 and 5 the cart can be sent as a quote request:
`POST /store-api/v1/checkout/quote-request`.

### Login, passkeys and password recovery

- Password login: `POST /store-api/v1/account/login` with `username` (the
  e-mail address) and `password`. The response header carries the session
  token.
- Passkey login: `POST /store-api/v1/account/passkey/login-options` returns
  WebAuthn request options and a `ceremonyId`. Pass `publicOptions` to the
  platform passkey API and send the result with `ceremonyId` to
  `POST /store-api/v1/account/passkey/login`. The ceremony is valid for five
  minutes and can be used once. Passkeys are bound to an address (`rpId`);
  apps send no `rpId` and use the shop address.
- Add a passkey: confirm the password with
  `POST /store-api/v1/account/passkey/register-options`, pass the options to
  the authenticator, and send the result with `ceremonyId` to
  `POST /store-api/v1/account/passkey/register`.
- Forgotten password: `POST /store-api/v1/account/recovery-password` sends a
  mail. The link contains a `hash`; check it with
  `POST /store-api/v1/account/customer-recovery-is-expired` if desired, then set
  the new password with
  `POST /store-api/v1/account/recovery-password-confirm`. A link is valid for
  two hours and only once; requesting a new mail invalidates the previous link.
- Account activation: the activation mail points to
  `https://shop.strombauer.com/showactivatecustomer/{code}`. Send the code as
  `hash` with the first password and `termsAccepted: true` to
  `POST /store-api/v1/account/activate`. The code works only once and the
  response logs the customer in.

Treat activation codes and recovery hashes like passwords: do not log them or
pass them to analytics.

### Rate limits and cart locking

The API answers `429` with `FRAMEWORK__RATE_LIMIT_EXCEEDED` when a limit is
exceeded, for example 30 e-mail checks per minute per IP address, 10
registration attempts per hour per IP address, or 10 passkey option requests
per minute. `409 CHECKOUT__CART_LOCKED` means another request is changing the
cart; send the request again shortly. The Go client does not retry
automatically.

### Prices and locked customers

Prices and stock are only returned for customers that are unlocked for orders.
Guests and customers in a locked customer group (for example *Blockiert*) get
the same product data with `calculatedPrice` empty (`[]`), `availableStock` and
`stockForecast` set to `null`, no `price` sort order, and stored carts without
prices (`available: false`).

Show ordering only when a product has a price. Adding a product as a locked
customer answers `403` with `CHECKOUT__CUSTOMER_PURCHASE_NOT_ALLOWED`; a cart
filled before the lock keeps its positions, shows a notice with this code, and
the order is rejected with it (`400`). `GET /store-api/v1/account/customer`
returns `customerState` (`LOCKED` or `UNLOCKED`); together with the session
state it distinguishes the three cases.

Guests and customers without price access receive
`400 STROMBAUER_STORE_API__PARAMETER_NOT_ALLOWED` when they send search
parameters such as `filter`, `limit`, `sort` or `term` to endpoints that do not
allow them.

### Search and clicks

1. Search with `GET /store-api/v1/search`: use `source=suggest` for preview
   requests while typing and `source=results` for the result page. Keep the
   `searchLogId` from the first page.
2. When a user opens a product from the results or adds it to the cart there,
   call `POST /store-api/v1/search-log/click/{searchLogId}`.

Searches without `source` are not recorded and return `searchLogId: null`.
The CLI's `strombauer search` sends `source=results` by default; pass
`--source ''` for an unrecorded search, for example during read-only checks.

## Error handling

Every error response has the same shape, a list with one entry per error:

```json
{
  "errors": [
    {
      "status": "400",
      "code": "VIOLATION::IS_BLANK_ERROR",
      "detail": "Bitte gib die Telefonnummer an.",
      "source": { "pointer": "/phoneNumber" }
    }
  ]
}
```

- `status` is the HTTP status as a string.
- `code` is the stable, machine-readable error code. **Decide by `code`, never
  by `detail`.**
- `detail` is a German sentence for the customer and can be shown as it is.
- `source.pointer` names the field the error belongs to, for example
  `/phoneNumber` or `/billingAddress/street`.

The client turns non-2xx responses into `*strombauer.APIError`:

```go
var apiErr *strombauer.APIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode)
	for _, entry := range apiErr.Errors {
		pointer := ""
		if entry.Source != nil {
			pointer = entry.Source.Pointer
		}
		fmt.Println(entry.Code, entry.Detail, pointer)
	}
}

if strombauer.IsCode(err, "CHECKOUT__CART_LOCKED") {
	// The cart is being changed by another request; retry shortly.
}
```

## CLI

Set the access key once and log in:

```sh
export STROMBAUER_ACCESS_KEY="your-access-key"

strombauer config set-access-key "$STROMBAUER_ACCESS_KEY"
strombauer login --username max@example.com
```

The session token is written to the config file (mode `0600`) and sent
automatically. Useful commands:

```sh
strombauer customer
strombauer products list --limit 5
strombauer products get <productId>
strombauer search "battery" --json
strombauer cart show
strombauer cart add <productId> --quantity 2
strombauer cart update <lineItemId> --quantity 3
strombauer cart remove <lineItemId>
strombauer cart set-payment <paymentMethodId>
strombauer cart set-shipping <shippingMethodId>
strombauer cart order --accept-terms
strombauer orders list --filter OPEN_INVOICES
strombauer orders get <orderId>
strombauer logout
```

`strombauer cart show` prints each line item with the ID that `cart update`
and `cart remove` expect, its quantity, and its unit and total price.

Configuration and overrides:

| Setting | Flag | Environment | Config key |
|---|---|---|---|
| Config file | `--config` | — | — |
| Base URL | `--base-url` | `STROMBAUER_BASE_URL` | `baseUrl` |
| Access key | `--access-key` | `STROMBAUER_ACCESS_KEY` | `accessKey` |
| Context token | `--context-token` | `STROMBAUER_CONTEXT_TOKEN` | `contextToken` |
| Timeout | `--timeout` | — | — |
| JSON output | `--json` | — | — |

The default config path is `$XDG_CONFIG_HOME/strombauer/config.json` on Linux
and the platform equivalent elsewhere. `strombauer config show` prints the
effective values with the secrets masked.

## Generated SDK

`api/api.gen.go` is generated from the vendored OpenAPI document with
`oapi-codegen` v2.5.1:

```sh
make generate
```

The command runs `internal/specfix` first. The published document contains
constructs that `oapi-codegen` cannot turn into a usable SDK, and one
incomplete schema. `specfix` fixes them deterministically:

- `$ref` values that point below a component schema are hoisted into top-level
  components and rewritten.
- The invalid tuple form `"items": []` is replaced with an empty schema.
- Properties that differ only by letter case are reduced to one Go field.
- Response schemas with inline enums are hoisted so their Go enum types are
  declared.
- The cart line-item schema declares only `listPrice`. It is completed from the
  documented response example and hoisted to `CompactCartLineItem`, so
  `CompactCart.Items` exposes line-item IDs, quantities, prices and stock.

The normalization is covered by tests in `internal/specfix`. To refresh from
the live documentation, replace `spec/openapi.json` with the OpenAPI document
served by the docs page and run `make generate` again. Do not edit
`api/api.gen.go` by hand; change the specification or the wrapper instead.

The complete API surface is available through the generated client:

```go
generated := client.API()
response, err := generated.GetStoreApiStrombauerExportJobsWithResponse(ctx)
```

## Project layout

```
api/                 generated Go types and client
internal/specfix/    OpenAPI normalization before generation
internal/cli/        cobra commands, config file, output formatting
cmd/strombauer/      CLI entry point
cmd/specfix/         normalization entry point
spec/openapi.json    vendored OpenAPI 3.0 document
oapi-codegen.yaml    generator configuration
```

## Read-only E2E tests

The `e2e` package talks to the live shop and is excluded from normal test runs
by the `e2e` build tag. It performs only `GET` requests against catalog and
account endpoints, apart from the optional login and logout described below;
it never changes the cart, places orders, or writes shop data.

1. Copy the example environment file:

   ```sh
   cp .env.example .env
   ```

2. Fill in `.env`. `STROMBAUER_ACCESS_KEY` is the only required value:

   | Variable | Required | Used for |
   |---|---|---|
   | `STROMBAUER_ACCESS_KEY` | yes | `sw-access-key`; identifies the sales channel |
   | `STROMBAUER_BASE_URL` | no | shop base URL; defaults to `https://shop.strombauer.com` |
   | `STROMBAUER_CONTEXT_TOKEN` | no | existing session for strictly read-only account tests; the tests never log in or out when it is set |
   | `STROMBAUER_USERNAME` | no | shop account e-mail; used only when no context token is set |
   | `STROMBAUER_PASSWORD` | no | shop account password; used only when no context token is set |

3. Run the tests:

   ```sh
   make test-e2e
   ```

`make test-e2e` loads `.env` automatically. Without `STROMBAUER_ACCESS_KEY` the
tests skip instead of failing. For strictly read-only authenticated checks, set
`STROMBAUER_CONTEXT_TOKEN` and leave the username/password empty. If you use
the username/password instead, the test logs in, runs the reads, and logs out;
those two calls only create and destroy a session.

The `.env` file is gitignored. The CLI reads the same
`STROMBAUER_ACCESS_KEY`, `STROMBAUER_CONTEXT_TOKEN` and `STROMBAUER_BASE_URL`
variables; use `strombauer login` for the CLI login.

## Development

```sh
make test        # unit tests
make test-race   # unit tests with the race detector
make test-e2e    # read-only E2E tests against the live shop
make vet         # go vet
make fmt-check   # gofmt check
make build       # build ./bin/strombauer
make generate    # regenerate api/api.gen.go from spec/openapi.json
```

There is no CI configuration in this repository yet.

## License

MIT — Copyright (c) 2026 Marcel Heers <marcel@heers.it>. See `LICENSE`.
