package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/mheers/shop-strombauer-api-golang/api"
	"github.com/spf13/cobra"
)

func newProductsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "products",
		Short: "Browse the product catalog",
	}
	cmd.AddCommand(newProductsListCmd(app), newProductsGetCmd(app))
	return cmd
}

func newProductsListCmd(app *App) *cobra.Command {
	var (
		page         int
		limit        int
		category     string
		manufacturer string
		sortOrder    string
		tag          string
		numbers      string
		productIDs   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List products",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}

			params := &api.GetStoreApiProductSearchParams{}
			if page > 0 {
				value := api.CriteriaPage(page)
				params.Page = &value
			}
			if limit > 0 {
				value := api.CriteriaLimit(limit)
				params.Limit = &value
			}
			if category != "" {
				value := api.CriteriaCategoryId(category)
				params.CategoryId = &value
			}
			if manufacturer != "" {
				value := api.CriteriaManufacturerId(manufacturer)
				params.ManufacturerId = &value
			}
			if sortOrder != "" {
				value := api.GetStoreApiProductSearchParamsSort(sortOrder)
				params.Sort = &value
			}
			if tag != "" {
				params.Tag = &tag
			}
			if numbers != "" {
				params.Numbers = &numbers
			}
			if productIDs != "" {
				value := api.CriteriaProductIds(productIDs)
				params.ProductIds = &value
			}

			products, err := client.Products(cmd.Context(), params)
			if err != nil {
				return err
			}
			if app.json {
				return app.printJSON(products)
			}
			return app.printProductTable(products.Elements, products.Total)
		},
	}

	flags := cmd.Flags()
	flags.IntVar(&page, "page", 0, "page number, starting at 1")
	flags.IntVar(&limit, "limit", 0, "products per page")
	flags.StringVar(&category, "category", "", "category ID")
	flags.StringVar(&manufacturer, "manufacturer", "", "manufacturer ID")
	flags.StringVar(&sortOrder, "sort", "", "sort order, e.g. name, -name, price, -popularity")
	flags.StringVar(&tag, "tag", "", "product tag")
	flags.StringVar(&numbers, "numbers", "", "comma-separated product, EAN or manufacturer numbers")
	flags.StringVar(&productIDs, "product-ids", "", "comma-separated product IDs")
	return cmd
}

func newProductsGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <productId>",
		Short: "Show one product",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			product, err := client.Product(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if app.json {
				return app.printJSON(product)
			}

			writer := tabwriter.NewWriter(app.opts.Out, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintf(writer, "ID:\t%s\n", product.Id)
			_, _ = fmt.Fprintf(writer, "Number:\t%s\n", product.ProductNumber)
			_, _ = fmt.Fprintf(writer, "Name:\t%s\n", deref(product.Name))
			_, _ = fmt.Fprintf(writer, "Type:\t%s\n", product.ProductType)
			if product.Manufacturer != nil {
				_, _ = fmt.Fprintf(writer, "Manufacturer:\t%s\n", deref(product.Manufacturer.Name))
			}
			if product.Ean != nil {
				_, _ = fmt.Fprintf(writer, "EAN:\t%s\n", *product.Ean)
			}
			if product.Description != nil && *product.Description != "" {
				_, _ = fmt.Fprintf(writer, "Description:\t%s\n", *product.Description)
			}
			return writer.Flush()
		},
	}
}

func newSearchCmd(app *App) *cobra.Command {
	var (
		source string
		page   int
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "search <term>",
		Short: "Search products",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}

			term := api.CriteriaTerm(args[0])
			params := &api.GetStoreApiSearchParams{Term: &term}
			if source != "" {
				value := api.GetStoreApiSearchParamsSource(source)
				params.Source = &value
			}
			if page > 0 {
				value := api.CriteriaPage(page)
				params.Page = &value
			}
			if limit > 0 {
				params.Limit = &limit
			}

			result, err := client.Search(cmd.Context(), params)
			if err != nil {
				return err
			}
			if app.json {
				return app.printJSON(result)
			}
			return app.printProductTable(result.Elements, result.Total)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&source, "source", "results", "where the search was typed: results or suggest (empty to avoid recording)")
	flags.IntVar(&page, "page", 0, "page number, starting at 1")
	flags.IntVar(&limit, "limit", 0, "products per page")
	return cmd
}

func (a *App) printProductTable(products []api.StrombauerProduct, total int) error {
	writer := tabwriter.NewWriter(a.opts.Out, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "ID\tNUMBER\tTYPE\tNAME")
	for _, product := range products {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", product.Id, product.ProductNumber, product.ProductType, deref(product.Name))
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(a.opts.Out, "\n%d of %d products\n", len(products), total)
	return err
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
