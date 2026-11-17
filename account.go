package strombauer

import (
	"context"
	"errors"
	"strings"

	"github.com/mheers/shop-strombauer-api-golang/api"
)

// EmailCheckResult is the answer of the first registration step.
type EmailCheckResult struct {
	// Valid reports whether the address is syntactically valid.
	Valid bool

	// Mode is "login" for an existing customer, "register" for a new one, or
	// "" for an invalid address.
	Mode string
}

// Login logs in with the shop account's e-mail address and password. The
// session token from the response is stored on the client automatically.
func (c *Client) Login(ctx context.Context, username, password string) error {
	if strings.TrimSpace(username) == "" {
		return errors.New("strombauer: username is required")
	}
	if password == "" {
		return errors.New("strombauer: password is required")
	}

	resp, err := c.api.PostStoreApiAccountLoginWithResponse(ctx, api.PostStoreApiAccountLoginJSONRequestBody{
		Username: username,
		Password: password,
	})
	if err != nil {
		return wrapRequestError(err)
	}
	return responseError(resp.StatusCode(), resp.Body)
}

// Logout ends the current session. The shop returns a new anonymous context
// token, which is stored on the client automatically.
func (c *Client) Logout(ctx context.Context) error {
	resp, err := c.api.PostStoreApiAccountLogoutWithResponse(ctx)
	if err != nil {
		return wrapRequestError(err)
	}
	return responseError(resp.StatusCode(), resp.Body)
}

// Customer returns the account data of the logged-in customer.
func (c *Client) Customer(ctx context.Context) (*api.AccountCustomerResponse, error) {
	resp, err := c.api.GetStoreApiAccountCustomerWithResponse(ctx)
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

// EmailCheck tells whether an account exists for the e-mail address.
func (c *Client) EmailCheck(ctx context.Context, email string) (*EmailCheckResult, error) {
	if strings.TrimSpace(email) == "" {
		return nil, errors.New("strombauer: email is required")
	}

	resp, err := c.api.PostStoreApiStrombauerAccountEmailCheckWithResponse(ctx, api.PostStoreApiStrombauerAccountEmailCheckJSONRequestBody{
		Email: email,
	})
	if err != nil {
		return nil, wrapRequestError(err)
	}
	if err := responseError(resp.StatusCode(), resp.Body); err != nil {
		return nil, wrapRequestError(err)
	}
	if resp.JSON200 == nil {
		return nil, unexpectedResponse(resp.StatusCode(), resp.Body)
	}

	result := &EmailCheckResult{Valid: resp.JSON200.Valid}
	if resp.JSON200.Mode != nil {
		result.Mode = string(*resp.JSON200.Mode)
	}
	return result, nil
}

// ChangePassword changes the password of the logged-in customer.
func (c *Client) ChangePassword(ctx context.Context, currentPassword, newPassword string) error {
	resp, err := c.api.PostStoreApiAccountChangePasswordWithResponse(ctx, api.PostStoreApiAccountChangePasswordJSONRequestBody{
		Password:           currentPassword,
		NewPassword:        newPassword,
		NewPasswordConfirm: newPassword,
	})
	if err != nil {
		return wrapRequestError(err)
	}
	return responseError(resp.StatusCode(), resp.Body)
}

// RequestPasswordRecovery sends a password recovery mail.
func (c *Client) RequestPasswordRecovery(ctx context.Context, email string) error {
	resp, err := c.api.PostStoreApiAccountRecoverySendMailWithResponse(ctx, api.PostStoreApiAccountRecoverySendMailJSONRequestBody{
		Email: email,
	})
	if err != nil {
		return wrapRequestError(err)
	}
	return responseError(resp.StatusCode(), resp.Body)
}

// ConfirmPasswordRecovery sets a new password with the hash from the recovery
// mail.
func (c *Client) ConfirmPasswordRecovery(ctx context.Context, hash, newPassword string) error {
	resp, err := c.api.PostStoreApiAccountRecoveryPasswordWithResponse(ctx, api.PostStoreApiAccountRecoveryPasswordJSONRequestBody{
		Hash:               hash,
		NewPassword:        newPassword,
		NewPasswordConfirm: newPassword,
	})
	if err != nil {
		return wrapRequestError(err)
	}
	return responseError(resp.StatusCode(), resp.Body)
}

// Activate activates an inactive account with the code from its activation
// mail and accepts the terms and conditions.
func (c *Client) Activate(ctx context.Context, hash, password string) error {
	resp, err := c.api.PostStoreApiStrombauerAccountActivateWithResponse(ctx, api.PostStoreApiStrombauerAccountActivateJSONRequestBody{
		Hash:          hash,
		Password:      password,
		TermsAccepted: true,
	})
	if err != nil {
		return wrapRequestError(err)
	}
	return responseError(resp.StatusCode(), resp.Body)
}
