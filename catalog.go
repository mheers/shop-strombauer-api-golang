package strombauer

import (
	"context"

	"github.com/mheers/shop-strombauer-api-golang/api"
)

// Products returns a page of the product list.
func (c *Client) Products(ctx context.Context, params *api.GetStoreApiProductSearchParams) (*api.ProductListResponse, error) {
	resp, err := c.api.GetStoreApiProductSearchWithResponse(ctx, params)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Product returns the detail page of one product.
func (c *Client) Product(ctx context.Context, productID string) (*api.ProductDetailResponse, error) {
	resp, err := c.api.GetStoreApiProductDetailWithResponse(ctx, productID)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Search runs a product search. Set Source to "results" or "suggest" so the
// shop records the search and returns a searchLogId.
func (c *Client) Search(ctx context.Context, params *api.GetStoreApiSearchParams) (*api.ProductSearchResponse, error) {
	resp, err := c.api.GetStoreApiSearchWithResponse(ctx, params)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Categories returns the category tree.
func (c *Client) Categories(ctx context.Context, params *api.GetStoreApiCategorySearchParams) (*api.CategoryListResponse, error) {
	resp, err := c.api.GetStoreApiCategorySearchWithResponse(ctx, params)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Countries returns the countries a customer can register or ship to.
func (c *Client) Countries(ctx context.Context) (*api.CountryListResponse, error) {
	resp, err := c.api.GetStoreApiCountryWithResponse(ctx)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Manufacturers returns a page of manufacturers.
func (c *Client) Manufacturers(ctx context.Context, params *api.GetStoreApiStrombauerManufacturersParams) (*api.StrombauerManufacturerListResponse, error) {
	resp, err := c.api.GetStoreApiStrombauerManufacturersWithResponse(ctx, params)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Manufacturer returns one manufacturer.
func (c *Client) Manufacturer(ctx context.Context, manufacturerID string) (*api.StrombauerManufacturerDetail, error) {
	resp, err := c.api.GetStoreApiStrombauerManufacturerWithResponse(ctx, manufacturerID)
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}
