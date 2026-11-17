// Package cli implements the strombauer command line client on top of the
// reusable strombauer package.
package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mheers/shop-strombauer-api-golang"
	"github.com/spf13/cobra"
)

// Options injects the process environment so the commands can be tested.
type Options struct {
	In         io.Reader
	Out        io.Writer
	Err        io.Writer
	ConfigPath string
	Getenv     func(string) string
}

// App holds the state shared by all commands.
type App struct {
	opts Options

	configPath   string
	baseURL      string
	accessKey    string
	contextToken string
	timeout      time.Duration
	json         bool

	config    *Config
	apiClient *strombauer.Client
	input     *bufio.Reader
}

// NewRootCmd builds the cobra command tree.
func NewRootCmd(opts Options) *cobra.Command {
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}
	if opts.Getenv == nil {
		opts.Getenv = os.Getenv
	}

	app := &App{opts: opts}
	defaultConfigPath := opts.ConfigPath
	if defaultConfigPath == "" {
		if path, err := DefaultConfigPath(); err == nil {
			defaultConfigPath = path
		}
	}

	root := &cobra.Command{
		Use:   "strombauer",
		Short: "Command line client for the Strombauer Shopware Store API",
		Long: `strombauer talks to the Strombauer shop's Store API.

The access key identifies the sales channel and is required for every request.
Set it with --access-key, STROMBAUER_ACCESS_KEY or "strombauer config set-access-key".
Log in once with "strombauer login"; the session token is stored in the config
file and sent automatically afterwards.`,
		Version:       strombauer.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(opts.In)
	root.SetOut(opts.Out)
	root.SetErr(opts.Err)
	root.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		return app.persistContextToken()
	}

	flags := root.PersistentFlags()
	flags.StringVar(&app.configPath, "config", defaultConfigPath, "path to the config file")
	flags.StringVar(&app.baseURL, "base-url", "", "shop base URL (env STROMBAUER_BASE_URL)")
	flags.StringVar(&app.accessKey, "access-key", "", "sw-access-key (env STROMBAUER_ACCESS_KEY)")
	flags.StringVar(&app.contextToken, "context-token", "", "sw-context-token (env STROMBAUER_CONTEXT_TOKEN)")
	flags.DurationVar(&app.timeout, "timeout", 30*time.Second, "HTTP request timeout")
	flags.BoolVar(&app.json, "json", false, "print the raw JSON response")

	root.AddCommand(
		newLoginCmd(app),
		newLogoutCmd(app),
		newCustomerCmd(app),
		newProductsCmd(app),
		newSearchCmd(app),
		newCartCmd(app),
		newOrdersCmd(app),
		newConfigCmd(app),
		newVersionCmd(app),
	)

	return root
}

func (a *App) inputReader() *bufio.Reader {
	if a.input == nil {
		a.input = bufio.NewReader(a.opts.In)
	}
	return a.input
}

func (a *App) getenv(key string) string {
	if a.opts.Getenv == nil {
		return ""
	}
	return a.opts.Getenv(key)
}

func (a *App) loadConfig() (*Config, error) {
	if a.config != nil {
		return a.config, nil
	}
	if a.configPath == "" {
		return nil, errors.New("strombauer: no config path; pass --config")
	}

	config, err := LoadConfig(a.configPath)
	if err != nil {
		return nil, err
	}
	a.config = config
	return config, nil
}

func (a *App) client() (*strombauer.Client, error) {
	if a.apiClient != nil {
		return a.apiClient, nil
	}

	config, err := a.loadConfig()
	if err != nil {
		return nil, err
	}

	accessKey := firstNonEmpty(
		a.accessKey,
		a.getenv("STROMBAUER_ACCESS_KEY"),
		config.AccessKey,
	)
	if accessKey == "" {
		return nil, errors.New("strombauer: access key missing; use --access-key, STROMBAUER_ACCESS_KEY or \"strombauer config set-access-key\"")
	}

	baseURL := firstNonEmpty(
		a.baseURL,
		a.getenv("STROMBAUER_BASE_URL"),
		config.BaseURL,
		strombauer.DefaultBaseURL,
	)
	contextToken := firstNonEmpty(
		a.contextToken,
		a.getenv("STROMBAUER_CONTEXT_TOKEN"),
		config.ContextToken,
	)

	options := []strombauer.Option{strombauer.WithBaseURL(baseURL)}
	if contextToken != "" {
		options = append(options, strombauer.WithContextToken(contextToken))
	}
	if a.timeout > 0 {
		options = append(options, strombauer.WithHTTPClient(&http.Client{Timeout: a.timeout}))
	}

	client, err := strombauer.New(accessKey, options...)
	if err != nil {
		return nil, err
	}
	a.apiClient = client
	return client, nil
}

func (a *App) saveContextToken(token string) error {
	config, err := a.loadConfig()
	if err != nil {
		return err
	}
	config.ContextToken = token
	if err := SaveConfig(a.configPath, config); err != nil {
		return err
	}
	a.config = config
	return nil
}

// persistContextToken stores a token that the shop rotated during the last
// request, so the next invocation continues the same session.
func (a *App) persistContextToken() error {
	if a.apiClient == nil {
		return nil
	}
	token := a.apiClient.ContextToken()
	if token == "" {
		return nil
	}
	config, err := a.loadConfig()
	if err != nil {
		return err
	}
	if config.ContextToken == token {
		return nil
	}
	return a.saveContextToken(token)
}

func (a *App) saveAccessKey(accessKey string) error {
	config, err := a.loadConfig()
	if err != nil {
		return err
	}
	config.AccessKey = accessKey
	if err := SaveConfig(a.configPath, config); err != nil {
		return err
	}
	a.config = config
	return nil
}

func (a *App) saveBaseURL(baseURL string) error {
	config, err := a.loadConfig()
	if err != nil {
		return err
	}
	config.BaseURL = baseURL
	if err := SaveConfig(a.configPath, config); err != nil {
		return err
	}
	a.config = config
	return nil
}

func (a *App) printJSON(value any) error {
	encoder := json.NewEncoder(a.opts.Out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mask(secret string) string {
	if secret == "" {
		return "(not set)"
	}
	if len(secret) <= 8 {
		return "********"
	}
	return "********" + secret[len(secret)-4:]
}

func newVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the client version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(app.opts.Out, "strombauer %s\n", strombauer.Version)
			return err
		},
	}
}
