package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/mheers/shop-strombauer-api-golang/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/spf13/cobra"
)

func newOrdersCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orders",
		Short: "Browse the customer's sales orders",
	}
	cmd.AddCommand(newOrdersListCmd(app), newOrdersGetCmd(app))
	return cmd
}

func newOrdersListCmd(app *App) *cobra.Command {
	var (
		term   string
		filter []string
		page   int
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sales orders",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}

			params := &api.GetStoreApiStrombauerSalesOrdersParams{}
			if term != "" {
				params.Term = &term
			}
			if len(filter) > 0 {
				values := make([]api.GetStoreApiStrombauerSalesOrdersParamsFilter, 0, len(filter))
				for _, value := range filter {
					values = append(values, api.GetStoreApiStrombauerSalesOrdersParamsFilter(value))
				}
				params.Filter = &values
			}
			if page > 0 {
				params.P = &page
			}
			if limit > 0 {
				params.Limit = &limit
			}

			orders, err := client.Orders(cmd.Context(), params)
			if err != nil {
				return err
			}
			if app.json {
				return app.printJSON(orders)
			}

			writer := tabwriter.NewWriter(app.opts.Out, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(writer, "ID\tNUMBER\tDATE\tREFERENCE\tEXTERNAL")
			for _, order := range orders.Items {
				_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
					deref(order.Id),
					deref(order.Number),
					formatDate(order.OrderDate),
					deref(order.Reference),
					deref(order.ExternalOrderNumber),
				)
			}
			if err := writer.Flush(); err != nil {
				return err
			}

			_, err = fmt.Fprintf(app.opts.Out,
				"\n%d orders, %d pending; open invoices: %d, open deliveries: %d, action required: %d\n",
				len(orders.Items),
				orders.PendingOrderCount,
				orders.FilterCounts.OPENINVOICES,
				orders.FilterCounts.OPENDELIVERIES,
				orders.FilterCounts.ACTIONREQUIRED,
			)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&term, "term", "", "search term")
	flags.StringSliceVar(&filter, "filter", nil, "filters: OPEN_INVOICES, OPEN_DELIVERIES, ACTION_REQUIRED")
	flags.IntVar(&page, "page", 0, "page number, starting at 1")
	flags.IntVar(&limit, "limit", 0, "orders per page")
	return cmd
}

func newOrdersGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <orderId>",
		Short: "Show one sales order",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			order, err := client.Order(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if app.json {
				return app.printJSON(order)
			}

			writer := tabwriter.NewWriter(app.opts.Out, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintf(writer, "ID:\t%s\n", deref(order.Id))
			if order.Sections != nil {
				for _, section := range *order.Sections {
					status := "-"
					if section.Status != nil && section.Status.Name != nil {
						status = *section.Status.Name
					}
					marker := ""
					if section.Current {
						marker = " (current)"
					}
					_, _ = fmt.Fprintf(writer, "%s:\t%s%s\n", section.Name, status, marker)
				}
			}
			return writer.Flush()
		},
	}
}

func formatDate(date *openapi_types.Date) string {
	if date == nil {
		return ""
	}
	return date.Format("2006-01-02")
}
