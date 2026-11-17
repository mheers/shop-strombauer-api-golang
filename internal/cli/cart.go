package cli

import (
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/mheers/shop-strombauer-api-golang/api"
	"github.com/spf13/cobra"
)

func newCartCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cart",
		Short: "Inspect and change the shopping cart",
	}
	cmd.AddCommand(
		newCartShowCmd(app),
		newCartAddCmd(app),
		newCartUpdateCmd(app),
		newCartRemoveCmd(app),
		newCartClearCmd(app),
		newCartSetPaymentCmd(app),
		newCartSetShippingCmd(app),
		newCartOrderCmd(app),
	)
	return cmd
}

func newCartShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the current cart",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			cart, err := client.Cart(cmd.Context())
			if err != nil {
				return err
			}
			return app.printCart(cart)
		},
	}
}

func newCartAddCmd(app *App) *cobra.Command {
	var quantity int

	cmd := &cobra.Command{
		Use:   "add <productId>",
		Short: "Add a product to the cart",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			cart, err := client.AddProduct(cmd.Context(), args[0], quantity)
			if err != nil {
				return err
			}
			return app.printCart(cart)
		},
	}
	cmd.Flags().IntVar(&quantity, "quantity", 1, "quantity to add")
	return cmd
}

func newCartUpdateCmd(app *App) *cobra.Command {
	var quantity int

	cmd := &cobra.Command{
		Use:   "update <lineItemId>",
		Short: "Change the quantity of a cart position",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			cart, err := client.UpdateCartItem(cmd.Context(), args[0], quantity)
			if err != nil {
				return err
			}
			return app.printCart(cart)
		},
	}
	cmd.Flags().IntVar(&quantity, "quantity", 0, "new quantity")
	_ = cmd.MarkFlagRequired("quantity")
	return cmd
}

func newCartRemoveCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <lineItemId>",
		Short: "Remove a position from the cart",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			cart, err := client.RemoveCartItem(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if cart == nil {
				_, err := fmt.Fprintln(app.opts.Out, "Position removed.")
				return err
			}
			return app.printCart(cart)
		},
	}
}

func newCartClearCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all positions from the cart",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			if err := client.ClearCart(cmd.Context()); err != nil {
				return err
			}
			_, err = fmt.Fprintln(app.opts.Out, "Cart cleared.")
			return err
		},
	}
}

func newCartSetPaymentCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "set-payment <paymentMethodId>",
		Short: "Select the payment method",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			cart, err := client.SetPaymentMethod(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return app.printCart(cart)
		},
	}
}

func newCartSetShippingCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "set-shipping <shippingMethodId>",
		Short: "Select the shipping method",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			cart, err := client.SetShippingMethod(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return app.printCart(cart)
		},
	}
}

func newCartOrderCmd(app *App) *cobra.Command {
	var acceptTerms bool

	cmd := &cobra.Command{
		Use:   "order",
		Short: "Place the order",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !acceptTerms {
				return errors.New("strombauer: pass --accept-terms to place the order")
			}
			client, err := app.client()
			if err != nil {
				return err
			}
			number, err := client.PlaceOrder(cmd.Context(), true)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(app.opts.Out, "Order placed: %s\n", number)
			return err
		},
	}
	cmd.Flags().BoolVar(&acceptTerms, "accept-terms", false, "accept the terms and conditions")
	return cmd
}

func (a *App) printCart(cart *api.CompactCart) error {
	if a.json {
		return a.printJSON(cart)
	}

	writer := tabwriter.NewWriter(a.opts.Out, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(writer, "Items:\t%d\n", len(cart.Items))
	_, _ = fmt.Fprintf(writer, "Net:\t%s\n", formatAmount(cart.Totals.Net))
	_, _ = fmt.Fprintf(writer, "Gross:\t%s\n", formatAmount(cart.Totals.Gross))
	if cart.Totals.Savings != nil {
		_, _ = fmt.Fprintf(writer, "Savings:\t%s\n", formatAmount(cart.Totals.Savings))
	}
	if cart.Totals.TaxExemption != nil {
		_, _ = fmt.Fprintf(writer, "Tax exemption:\t%s\n", cart.Totals.TaxExemption.Message)
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	for _, notice := range cart.Notices {
		if _, err := fmt.Fprintf(a.opts.Out, "Notice [%s]: %s\n", notice.Code, notice.Message); err != nil {
			return err
		}
	}
	return nil
}

func formatAmount(amount *float32) string {
	if amount == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f EUR", *amount)
}
