package cli

import (
	"fmt"
	"net/url"
	"text/tabwriter"

	"github.com/mheers/shop-strombauer-api-golang"
	"github.com/spf13/cobra"
)

func newConfigCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show and change the stored configuration",
	}
	cmd.AddCommand(
		newConfigShowCmd(app),
		newConfigSetAccessKeyCmd(app),
		newConfigSetBaseURLCmd(app),
	)
	return cmd
}

func newConfigShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the effective configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := app.loadConfig()
			if err != nil {
				return err
			}

			writer := tabwriter.NewWriter(app.opts.Out, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintf(writer, "Config:\t%s\n", app.configPath)
			_, _ = fmt.Fprintf(writer, "Base URL:\t%s\n", firstNonEmpty(app.baseURL, app.getenv("STROMBAUER_BASE_URL"), config.BaseURL, strombauer.DefaultBaseURL))
			_, _ = fmt.Fprintf(writer, "Access key:\t%s\n", mask(firstNonEmpty(app.accessKey, app.getenv("STROMBAUER_ACCESS_KEY"), config.AccessKey)))
			_, _ = fmt.Fprintf(writer, "Context token:\t%s\n", mask(firstNonEmpty(app.contextToken, app.getenv("STROMBAUER_CONTEXT_TOKEN"), config.ContextToken)))
			return writer.Flush()
		},
	}
}

func newConfigSetAccessKeyCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "set-access-key <key>",
		Short: "Store the sw-access-key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.saveAccessKey(args[0]); err != nil {
				return err
			}
			_, err := fmt.Fprintf(app.opts.Out, "Access key stored in %s.\n", app.configPath)
			return err
		},
	}
}

func newConfigSetBaseURLCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "set-base-url <url>",
		Short: "Store the shop base URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parsed, err := url.Parse(args[0])
			if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return fmt.Errorf("strombauer: invalid base URL %q", args[0])
			}
			if err := app.saveBaseURL(args[0]); err != nil {
				return err
			}
			_, err = fmt.Fprintf(app.opts.Out, "Base URL stored in %s.\n", app.configPath)
			return err
		},
	}
}
