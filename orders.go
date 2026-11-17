package strombauer

import (
	"context"

	"github.com/mheers/shop-strombauer-api-golang/api"
)

// Orders returns the customer's sales orders.
func (c *Client) Orders(ctx context.Context, params *api.GetStoreApiStrombauerSalesOrdersParams) (*api.BcOrdersResponse, error) {
	resp, err := c.api.GetStoreApiStrombauerSalesOrdersWithResponse(ctx, params)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, wrapRequestError(err)
	}
	if resp.JSON200 == nil {
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Order returns one sales order with its documents and delivery state.
func (c *Client) Order(ctx context.Context, orderID string) (*api.BcOrderDetailResponse, error) {
	resp, err := c.api.GetStoreApiStrombauerSalesOrderDetailWithResponse(ctx, orderID)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, wrapRequestError(err)
	}
	if resp.JSON200 == nil {
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}
