package strombauer

import (
	"context"
	"errors"
	"strings"

	"github.com/mheers/shop-strombauer-api-golang/api"
)

// ErrEmptyCart is returned when AddToCart is called without items.
var ErrEmptyCart = errors.New("strombauer: no cart items given")

// ErrInvalidQuantity is returned when a cart quantity is less than 1. The
// Store API requires a minimum quantity of 1 for every line item.
var ErrInvalidQuantity = errors.New("strombauer: quantity must be at least 1")

// ErrTermsNotAccepted is returned by PlaceOrder when termsAccepted is false.
var ErrTermsNotAccepted = errors.New("strombauer: terms and conditions must be accepted")

// Cart returns the current cart of the session.
func (c *Client) Cart(ctx context.Context) (*api.CompactCart, error) {
	resp, err := c.api.GetStoreApiCheckoutCartReadWithResponse(ctx)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// AddToCart adds one or more line items and returns the updated cart.
func (c *Client) AddToCart(ctx context.Context, items ...api.CartProductLineItem) (*api.CompactCart, error) {
	if len(items) == 0 {
		return nil, ErrEmptyCart
	}
	for i := range items {
		if strings.TrimSpace(items[i].ProductId) == "" {
			return nil, errors.New("strombauer: cart item has no product ID")
		}
		if items[i].Quantity != nil && *items[i].Quantity < 1 {
			return nil, ErrInvalidQuantity
		}
	}

	resp, err := c.api.PostStoreApiCheckoutCartAddWithResponse(ctx, api.CartItems{Items: items})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// AddProduct adds a product to the cart. A quantity of zero or less means 1.
func (c *Client) AddProduct(ctx context.Context, productID string, quantity int) (*api.CompactCart, error) {
	if quantity <= 0 {
		quantity = 1
	}
	quantityValue := quantity
	return c.AddToCart(ctx, api.CartProductLineItem{
		ProductId: productID,
		Quantity:  &quantityValue,
	})
}

// UpdateCartItem sets the quantity of an existing line item. The quantity must
// be at least 1.
func (c *Client) UpdateCartItem(ctx context.Context, lineItemID string, quantity int) (*api.CompactCart, error) {
	if quantity < 1 {
		return nil, ErrInvalidQuantity
	}
	resp, err := c.api.PatchStoreApiStrombauerCheckoutCartUpdateItemWithResponse(ctx, lineItemID, api.PatchStoreApiStrombauerCheckoutCartUpdateItemJSONRequestBody{
		Quantity: quantity,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// RemoveCartItem removes a line item from the cart. The returned cart is nil
// when the shop answers without a body.
func (c *Client) RemoveCartItem(ctx context.Context, lineItemID string) (*api.CompactCart, error) {
	resp, err := c.api.DeleteStoreApiStrombauerCheckoutCartRemoveItemWithResponse(ctx, lineItemID)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, err
	}
	return resp.JSON200, nil
}

// ClearCart removes every line item from the cart.
func (c *Client) ClearCart(ctx context.Context) error {
	resp, err := c.api.DeleteStoreApiCheckoutCartDeleteWithResponse(ctx)
	if err != nil {
		return wrapRequestError(err)
	}
	return responseError(resp.StatusCode(), resp.Body)
}

// SetPaymentMethod selects the payment method of the cart.
func (c *Client) SetPaymentMethod(ctx context.Context, paymentMethodID string) (*api.CompactCart, error) {
	resp, err := c.api.PostStoreApiStrombauerCheckoutSetPaymentMethodWithResponse(ctx, api.PostStoreApiStrombauerCheckoutSetPaymentMethodJSONRequestBody{
		PaymentMethodId: paymentMethodID,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// SetShippingMethod selects the shipping method of the cart.
func (c *Client) SetShippingMethod(ctx context.Context, shippingMethodID string) (*api.CompactCart, error) {
	resp, err := c.api.PostStoreApiStrombauerCheckoutSetShippingMethodWithResponse(ctx, api.PostStoreApiStrombauerCheckoutSetShippingMethodJSONRequestBody{
		ShippingMethodId: shippingMethodID,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// SetShippingAddress selects the shipping address of the cart.
func (c *Client) SetShippingAddress(ctx context.Context, shippingAddressID string) (*api.CompactCart, error) {
	resp, err := c.api.PostStoreApiStrombauerCheckoutSetShippingAddressWithResponse(ctx, api.PostStoreApiStrombauerCheckoutSetShippingAddressJSONRequestBody{
		ShippingAddressId: shippingAddressID,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// SetShippingDate selects the requested shipping date. The format is
// "Y-m-d H:i:s" or "Y-m-d", depending on the shipping method.
func (c *Client) SetShippingDate(ctx context.Context, datetime string) (*api.CompactCart, error) {
	resp, err := c.api.PostStoreApiStrombauerCheckoutSetShippingDateWithResponse(ctx, api.PostStoreApiStrombauerCheckoutSetShippingDateJSONRequestBody{
		Datetime: datetime,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// SetShippingOption selects the shipping option of the cart.
func (c *Client) SetShippingOption(ctx context.Context, shippingOptionID string) (*api.CompactCart, error) {
	resp, err := c.api.PostStoreApiStrombauerCheckoutSetShippingOptionWithResponse(ctx, api.PostStoreApiStrombauerCheckoutSetShippingOptionJSONRequestBody{
		ShippingOptionId: shippingOptionID,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// SetShippingNotice sets the delivery notice of the cart.
func (c *Client) SetShippingNotice(ctx context.Context, notice string) (*api.CompactCart, error) {
	resp, err := c.api.PostStoreApiStrombauerCheckoutSetShippingNoticeWithResponse(ctx, api.PostStoreApiStrombauerCheckoutSetShippingNoticeJSONRequestBody{
		ShippingNotice: notice,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// SetPartialDelivery selects "partial" or "complete" delivery.
func (c *Client) SetPartialDelivery(ctx context.Context, partialDelivery string) (*api.CompactCart, error) {
	resp, err := c.api.PostStoreApiStrombauerCheckoutSetPartialDeliveryWithResponse(ctx, api.PostStoreApiStrombauerCheckoutSetPartialDeliveryJSONRequestBody{
		PartialDelivery: api.PostStoreApiStrombauerCheckoutSetPartialDeliveryJSONBodyPartialDelivery(partialDelivery),
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// SetShippingAddOn books or declines an optional shipping service.
func (c *Client) SetShippingAddOn(ctx context.Context, shippingAddOnID string, value bool) (*api.CompactCart, error) {
	resp, err := c.api.PostStoreApiStrombauerCheckoutSetShippingAddOnWithResponse(ctx, api.PostStoreApiStrombauerCheckoutSetShippingAddOnJSONRequestBody{
		ShippingAddOnId: shippingAddOnID,
		Value:           value,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// SetComment sets the customer comment of the cart.
func (c *Client) SetComment(ctx context.Context, comment string) (*api.CompactCart, error) {
	resp, err := c.api.PostStoreApiStrombauerCheckoutSetCommentWithResponse(ctx, api.PostStoreApiStrombauerCheckoutSetCommentJSONRequestBody{
		Comment: comment,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// SetReference sets the customer reference of the cart.
func (c *Client) SetReference(ctx context.Context, reference string) (*api.CompactCart, error) {
	resp, err := c.api.PostStoreApiStrombauerCheckoutSetReferenceWithResponse(ctx, api.PostStoreApiStrombauerCheckoutSetReferenceJSONRequestBody{
		Reference: reference,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	return compactCartResult(resp.StatusCode(), resp.Body, resp.JSON200)
}

// PlaceOrder places the order and returns the shop's order number.
func (c *Client) PlaceOrder(ctx context.Context, termsAccepted bool) (string, error) {
	if !termsAccepted {
		return "", ErrTermsNotAccepted
	}

	resp, err := c.api.PostStoreApiCheckoutCartOrderWithResponse(ctx, api.PostStoreApiCheckoutCartOrderJSONRequestBody{
		TermsAccepted: api.PostStoreApiCheckoutCartOrderJSONBodyTermsAccepted(true),
	})
	if err != nil {
		return "", wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return "", err
	}
	if resp.JSON200 == nil {
		return "", unexpectedResponse(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200.Number, nil
}

func compactCartResult(statusCode int, body []byte, cart *api.CompactCart) (*api.CompactCart, error) {
	if err := responseError(statusCode, body); err != nil {
		return nil, err
	}
	if cart == nil {
		return nil, unexpectedResponse(statusCode, body)
	}
	return cart, nil
}
