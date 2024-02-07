package pgxman

import (
	"fmt"

	"github.com/eiannone/keyboard"
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/config"
	"github.com/pgxman/pgxman/internal/oauth"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "auth",
		Short:  "Authenticate with registry",
		Hidden: true,
	}

	cmd.AddCommand(newAuthLoginCmd())

	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to a registry account",
		RunE:  runAuthLogin,
	}

	return cmd
}

func runAuthLogin(c *cobra.Command, args []string) error {
	cfg, err := config.Read()
	if err != nil {
		return err
	}

	flow, err := oauth.InitFlow(
		oauth.FlowParams{
			ClientID: cfg.OAuth.ClientID,
			Scopes:   []string{"openid", "write:pubish_package"},
			Endpoint: cfg.OAuth.Endpoint,
		},
	)
	if err != nil {
		return err
	}
	defer func() {
		_ = flow.Done()
	}()

	io := pgxman.NewStdIO()
	con, err := io.Prompt(
		"Press Enter to open pgxman.com in your browser...",
		nil,
		[]keyboard.Key{keyboard.KeyEnter},
	)
	if err != nil {
		return err
	}

	if !con {
		return nil
	}

	if err := browser.OpenURL(flow.BrowserURL()); err != nil {
		return err
	}

	token, err := flow.WaitForToken(c.Context())
	if err != nil {
		return err
	}

	fmt.Println(token)

	return nil

}
