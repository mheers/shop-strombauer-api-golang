package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newLoginCmd(app *App) *cobra.Command {
	var username, password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in with the shop account",
		Long: `Logs in with the e-mail address and password of the shop account and stores
the returned sw-context-token in the config file.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if strings.TrimSpace(username) == "" {
				username, err = app.readLine("E-mail: ")
				if err != nil {
					return err
				}
			}
			if password == "" {
				password, err = app.readPassword()
				if err != nil {
					return err
				}
			}

			client, err := app.client()
			if err != nil {
				return err
			}
			if err := client.Login(cmd.Context(), username, password); err != nil {
				return err
			}
			if err := app.saveContextToken(client.ContextToken()); err != nil {
				return err
			}

			_, err = fmt.Fprintf(app.opts.Out, "Logged in as %s.\n", username)
			return err
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "e-mail address of the shop account")
	cmd.Flags().StringVar(&password, "password", "", "password (prefer the interactive prompt)")
	return cmd
}

func newLogoutCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "End the current session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			if err := client.Logout(cmd.Context()); err != nil {
				return err
			}
			if err := app.saveContextToken(client.ContextToken()); err != nil {
				return err
			}
			_, err = fmt.Fprintln(app.opts.Out, "Logged out.")
			return err
		},
	}
}

func newCustomerCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "customer",
		Short: "Show the logged-in customer",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := app.client()
			if err != nil {
				return err
			}
			customer, err := client.Customer(cmd.Context())
			if err != nil {
				return err
			}
			if app.json {
				return app.printJSON(customer)
			}

			writer := tabwriter.NewWriter(app.opts.Out, 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintf(writer, "Name:\t%s %s\n", customer.Customer.FirstName, customer.Customer.LastName)
			if customer.Customer.Company != nil && *customer.Customer.Company != "" {
				_, _ = fmt.Fprintf(writer, "Company:\t%s\n", *customer.Customer.Company)
			}
			_, _ = fmt.Fprintf(writer, "E-mail:\t%s\n", customer.Customer.Email)
			_, _ = fmt.Fprintf(writer, "Customer number:\t%s\n", customer.Customer.CustomerNumber)
			_, _ = fmt.Fprintf(writer, "State:\t%s\n", customer.CustomerState)
			_, _ = fmt.Fprintf(writer, "Addresses:\t%d\n", len(customer.Addresses))
			return writer.Flush()
		},
	}
}

func (a *App) readLine(prompt string) (string, error) {
	if _, err := fmt.Fprint(a.opts.Out, prompt); err != nil {
		return "", err
	}

	line, err := a.inputReader().ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (a *App) readPassword() (string, error) {
	if file, ok := a.opts.In.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		if _, err := fmt.Fprint(a.opts.Out, "Password: "); err != nil {
			return "", err
		}
		password, err := term.ReadPassword(int(file.Fd()))
		_, _ = fmt.Fprintln(a.opts.Out)
		if err != nil {
			return "", err
		}
		return string(password), nil
	}

	password, err := a.readLine("Password: ")
	if err != nil {
		return "", err
	}
	if password == "" {
		return "", errors.New("strombauer: password is required")
	}
	return password, nil
}
