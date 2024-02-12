package pgxman

import (
	"fmt"
	"net/url"

	"github.com/eiannone/keyboard"
	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/auth"
	"github.com/pgxman/pgxman/internal/config"
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
	cmd.AddCommand(newAuthTokenCmd())

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

func newAuthTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print the authentication token for the registry",
		RunE:  runAuthToken,
	}

	return cmd
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Read()
	if err != nil {
		return err
	}

	u, err := url.ParseRequestURI(flagRegistryURL)
	if err != nil {
		return err
	}

	io := pgxman.NewStdIO()

	if err := auth.Login(
		cmd.Context(),
		auth.LoginOptions{
			Config:      cfg,
			RegistryURL: u,
			BeforeLogin: func(registryHostname, registryLoginURL string) (bool, error) {
				con, err := io.Prompt(
					fmt.Sprintf("Press Enter to log in to %s in your browser...", registryHostname),
					nil,
					[]keyboard.Key{keyboard.KeyEnter},
				)
				if err != nil {
					return false, err
				}
				if !con {
					return false, nil
				}

				if err := browser.OpenURL(registryLoginURL); err != nil {
					return false, err
				}

				return true, nil
			},
			AfterLogin: func(email string) error {
				fmt.Fprintf(io.Stdout, "Logged in as %s\n", email)
				return nil
			},
		},
	); err != nil {
		return err
	}

	return nil
}

func runAuthToken(cmd *cobra.Command, args []string) error {
	u, err := url.ParseRequestURI(flagRegistryURL)
	if err != nil {
		return err
	}

	token, err := auth.Token(u)
	if err != nil {
		return fmt.Errorf("could not get token. Did you login?")
	}

	fmt.Println(token)
	return nil
}
