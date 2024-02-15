package pgxman

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/eiannone/keyboard"
	"github.com/pgxman/pgxman/internal/auth"
	"github.com/pgxman/pgxman/internal/cmd/cmdutil"
	"github.com/pgxman/pgxman/internal/config"
	"github.com/pgxman/pgxman/internal/iostreams"
	"github.com/pgxman/pgxman/internal/log"
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
	cmd.AddCommand(newAuthSignupCmd())
	cmd.AddCommand(newAuthStatusCmd())
	cmd.AddCommand(newAuthTokenCmd())
	cmd.AddCommand(newAuthLogoutCmd())

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

func newAuthSignupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signup",
		Short: "Sign up a registry account",
		RunE:  runAuthSignup,
	}

	return cmd
}

func newAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of a registry account",
		RunE:  runAuthLogout,
	}

	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "View authentication status",
		RunE:  runAuthStatus,
	}

	return cmd
}

func newAuthTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Print the authentication token",
		RunE:  runAuthToken,
	}

	return cmd
}

func runAuthSignup(cmd *cobra.Command, args []string) error {
	return loginOrSignup(cmd.Context(), auth.SignupScreen)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	return loginOrSignup(cmd.Context(), auth.LoginScreen)
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	u, err := url.ParseRequestURI(flagRegistryURL)
	if err != nil {
		return err
	}

	logger := log.NewTextLogger()
	err = auth.Logout(u)
	if err != nil {
		logger.Debug("error logging out", "error", err)
		return fmt.Errorf("not logged in to %s", u.Host)
	}

	fmt.Printf("Logged out of %s\n", u.Host)

	return nil
}

func runAuthToken(cmd *cobra.Command, args []string) error {
	u, err := url.ParseRequestURI(flagRegistryURL)
	if err != nil {
		return err
	}

	logger := log.NewTextLogger()
	token, err := auth.Token(u)
	if err != nil {
		logger.Debug("error getting token from keyring", "error", err)
		return fmt.Errorf("no oauth token found for %s", u.Host)
	}

	fmt.Println(token)
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	u, err := url.ParseRequestURI(flagRegistryURL)
	if err != nil {
		return err
	}

	client, err := newReigstryClient()
	if err != nil {
		return err
	}

	logger := log.NewTextLogger()
	user, err := client.GetUser(cmd.Context())
	if err != nil {
		logger.Debug("error getting user", "error", err)
		return fmt.Errorf("you are not logged in to %s. To log in, run: `pgxman auth login`", u.Host)
	}

	fmt.Printf("Logged in to %s as %s\n", u.Host, user.Email)
	return nil
}

func loginOrSignup(ctx context.Context, screen auth.Screen) error {
	cfg, err := config.Read()
	if err != nil {
		return err
	}

	u, err := url.ParseRequestURI(flagRegistryURL)
	if err != nil {
		return err
	}

	var (
		io     = iostreams.NewIOStreams()
		logger = log.NewTextLogger()
	)

	if err := auth.Login(
		ctx,
		auth.LoginOptions{
			Config:      cfg,
			RegistryURL: u,
			Screen:      screen,
			BeforeLogin: func(registryHostname, registryLoginURL string) error {
				var promptMsg string
				switch screen {
				case auth.SignupScreen:
					promptMsg = fmt.Sprintf("Press Enter to sign up at %s in your browser...", registryHostname)
				default:
					promptMsg = fmt.Sprintf("Press Enter to log in to %s in your browser...", registryHostname)
				}

				if err := io.Prompt(
					promptMsg,
					nil,
					[]keyboard.Key{keyboard.KeyEnter},
				); err != nil {
					return err
				}

				if err := browser.OpenURL(registryLoginURL); err != nil {
					return err
				}

				return nil
			},
			AfterLogin: func(email string) error {
				fmt.Fprintf(io.Stdout, "Logged in as %s\n", email)
				return nil
			},
		},
	); err != nil {
		logger.Debug("error logging in", "error", err)
		if errors.Is(err, iostreams.ErrAbortPrompt) {
			return cmdutil.SilentError
		}

		return err
	}

	return nil
}
