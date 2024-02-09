package auth

import (
	"context"
	"fmt"
	"net/url"

	"github.com/pgxman/pgxman/internal/config"
	"github.com/pgxman/pgxman/internal/keyring"
	"github.com/pgxman/pgxman/internal/registry"
)

type LoginOptions struct {
	Config      *config.Config
	RegistryURL *url.URL

	BeforeLogin func(registryHostname, registryLoginURL string) (bool, error)
	AfterLogin  func(email string) error
}

func Login(ctx context.Context, opts LoginOptions) error {
	flow, err := InitFlow(
		FlowParams{
			ClientID: opts.Config.OAuth.ClientID,
			Scopes:   []string{"openid", "write:pubish_package"},
			Endpoint: opts.Config.OAuth.Endpoint,
		},
	)
	if err != nil {
		return err
	}
	defer func() {
		_ = flow.Done()
	}()

	con, err := opts.BeforeLogin(opts.RegistryURL.Host, flow.BrowserURL())
	if err != nil {
		return err
	}
	if !con {
		return nil
	}

	token, err := flow.WaitForToken(ctx)
	if err != nil {
		return err
	}

	client, err := registry.NewClient(opts.RegistryURL.String(), token)
	if err != nil {
		return err
	}

	user, err := client.GetUser(ctx)
	if err != nil {
		return err
	}

	if err := keyring.Set(keyringServiceName(opts.RegistryURL.Host), string(user.Email), token); err != nil {
		return err
	}

	return opts.AfterLogin(string(user.Email))
}

func keyringServiceName(hostname string) string {
	return fmt.Sprintf("pgxman:%s", hostname)
}
