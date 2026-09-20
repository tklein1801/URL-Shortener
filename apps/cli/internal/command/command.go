package command

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
	"url-shortener-cli/internal/apiclient"
	"url-shortener-cli/internal/config"
)

type API interface {
	Status(context.Context) error
	List(context.Context) (apiclient.ListResponse, error)
	Shorten(context.Context, string) (apiclient.Link, error)
	Delete(context.Context, string) (apiclient.Link, error)
	RedirectURL(string) (string, error)
}
type Dependencies struct {
	In          io.Reader
	Out, Err    io.Writer
	ReadToken   func(io.Reader) (string, error)
	OpenBrowser func(context.Context, string) error
	NewClient   func(config.Config, time.Duration) (API, error)
	Version     string
}

func readToken(in io.Reader) (string, error) {
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		if err != nil {
			return "", errors.New("cannot read token")
		}
		return string(b), nil
	}
	s, err := bufio.NewReader(io.LimitReader(in, 4097)).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", errors.New("cannot read token")
	}
	if len(s) > 4096 {
		return "", errors.New("token input too long")
	}
	return strings.TrimSuffix(strings.TrimSuffix(s, "\n"), "\r"), nil
}
func openBrowser(ctx context.Context, target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", target)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", target)
	}
	return cmd.Run()
}
func New(d Dependencies) *cobra.Command {
	if d.In == nil {
		d.In = os.Stdin
	}
	if d.Out == nil {
		d.Out = os.Stdout
	}
	if d.Err == nil {
		d.Err = os.Stderr
	}
	if d.ReadToken == nil {
		d.ReadToken = readToken
	}
	if d.OpenBrowser == nil {
		d.OpenBrowser = openBrowser
	}
	if d.Version == "" {
		d.Version = "dev"
	}
	if d.NewClient == nil {
		d.NewClient = func(c config.Config, timeout time.Duration) (API, error) {
			return apiclient.New(c.ServerURL, c.Token, &http.Client{Timeout: timeout})
		}
	}
	var custom string
	var jsonOutput bool
	var timeout time.Duration
	root := &cobra.Command{Use: "surl", Short: "Shorten and manage URLs", SilenceUsage: true, SilenceErrors: true, Args: cobra.NoArgs}
	root.SetIn(d.In)
	root.SetOut(d.Out)
	root.SetErr(d.Err)
	root.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
	root.PersistentFlags().StringVar(&custom, "config", "", "Configuration file path")
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Print machine-readable JSON")
	root.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "Network and browser timeout")
	load := func() (string, config.Config, error) {
		p, err := config.Path(custom)
		if err != nil {
			return "", config.Config{}, err
		}
		c, err := config.Load(p)
		return p, c, err
	}
	output := func(value any, plain string) error {
		if jsonOutput {
			return json.NewEncoder(d.Out).Encode(value)
		}
		_, err := fmt.Fprintln(d.Out, plain)
		return err
	}
	cfg := &cobra.Command{Use: "config", Short: "Manage local configuration", Args: cobra.NoArgs}
	cfg.AddCommand(&cobra.Command{Use: "set-server <url>", Short: "Set the server URL", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := config.ParseURL(args[0]); err != nil {
			return err
		}
		p, c, err := load()
		if err != nil {
			return err
		}
		c.ServerURL = args[0]
		if err := config.Save(p, c); err != nil {
			return err
		}
		return output(map[string]string{"status": "saved"}, "Server URL saved.")
	}})
	cfg.AddCommand(&cobra.Command{Use: "set-token", Short: "Read and store the token without terminal echo", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		p, c, err := load()
		if err != nil {
			return err
		}
		fmt.Fprint(d.Err, "Master token: ")
		token, err := d.ReadToken(d.In)
		fmt.Fprintln(d.Err)
		if err != nil {
			return errors.New("cannot read token")
		}
		if err := config.ValidateToken(token); err != nil {
			return err
		}
		c.Token = token
		if err := config.Save(p, c); err != nil {
			return err
		}
		return output(map[string]string{"status": "saved"}, "Token saved.")
	}})
	cfg.AddCommand(&cobra.Command{Use: "show", Short: "Show configuration with masked token", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		_, c, err := load()
		if err != nil {
			return err
		}
		if c.Token != "" {
			c.Token = "********"
		}
		return output(c, fmt.Sprintf("server_url: %s\ntoken: %s", c.ServerURL, c.Token))
	}})
	root.AddCommand(cfg)
	add := func(use, short string, count int, action func(context.Context, API, []string) error) {
		root.AddCommand(&cobra.Command{Use: use, Short: short, Args: cobra.ExactArgs(count), RunE: func(cmd *cobra.Command, args []string) error {
			if timeout <= 0 {
				return errors.New("timeout must be positive")
			}
			_, c, err := load()
			if err != nil {
				return err
			}
			client, err := d.NewClient(c, timeout)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			return action(ctx, client, args)
		}})
	}
	add("status", "Check availability, storage, and authentication", 0, func(ctx context.Context, c API, _ []string) error {
		if err := c.Status(ctx); err != nil {
			return err
		}
		return output(map[string]string{"status": "ok"}, "Server available; storage ready; authentication valid.")
	})
	add("shorten <url>", "Create a shortened URL", 1, func(ctx context.Context, c API, args []string) error {
		link, err := c.Shorten(ctx, args[0])
		if err != nil {
			return err
		}
		target, err := c.RedirectURL(link.ID)
		if err != nil {
			return err
		}
		return output(struct {
			ID       string `json:"id"`
			URL      string `json:"url"`
			ShortURL string `json:"short_url"`
		}{link.ID, link.URL, target}, link.ID+"\t"+target)
	})
	add("list", "List shortened URLs", 0, func(ctx context.Context, c API, _ []string) error {
		result, err := c.List(ctx)
		if err != nil {
			return err
		}
		if jsonOutput {
			return output(result, "")
		}
		ids := make([]string, 0, len(result.Data))
		for id := range result.Data {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if _, err := fmt.Fprintf(d.Out, "%s\t%s\n", id, result.Data[id]); err != nil {
				return err
			}
		}
		return nil
	})
	add("open <id>", "Open a redirect URL in the browser", 1, func(ctx context.Context, c API, args []string) error {
		target, err := c.RedirectURL(args[0])
		if err != nil {
			return err
		}
		openErr := d.OpenBrowser(ctx, target)
		if err := output(map[string]any{"url": target, "opened": openErr == nil}, target); err != nil {
			return err
		}
		if openErr != nil {
			return errors.New("browser launch failed; open the printed URL manually")
		}
		return nil
	})
	add("delete <id>", "Delete a shortened URL", 1, func(ctx context.Context, c API, args []string) error {
		link, err := c.Delete(ctx, args[0])
		if err != nil {
			return err
		}
		return output(link, "Deleted "+link.ID)
	})
	root.AddCommand(&cobra.Command{Use: "version", Short: "Print version", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return output(map[string]string{"version": d.Version}, d.Version)
	}})
	return root
}

// Execute uses exit code 1 for command, configuration, network, and API failures.
func Execute(d Dependencies, args []string) int {
	root := New(d)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(root.ErrOrStderr(), "Error:", err)
		return 1
	}
	return 0
}
