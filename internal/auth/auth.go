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
	Screen      Screen

	BeforeLogin func(registryHostname, registryLoginURL string) error
	AfterLogin  func(email string) error
}

func Login(ctx context.Context, opts LoginOptions) error {
	flow, err := InitFlow(
		FlowParams{
			ClientID: opts.Config.OAuth.ClientID,
			Scopes:   []string{"openid", "write:publish_extension"},
			Audience: opts.Config.OAuth.Audience,
			Endpoint: opts.Config.OAuth.Endpoint,
			Screen:   opts.Screen,
		},
	)
	if err != nil {
		return err
	}
	defer func() {
		_ = flow.Done()
	}()

	err = opts.BeforeLogin(opts.RegistryURL.Host, flow.BrowserURL())
	if err != nil {
		return err
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

	if err := keyring.Set(keyringServiceName(opts.RegistryURL.Host), "", token); err != nil {
		return err
	}

	return opts.AfterLogin(string(user.Email))
}

func Logout(registryURL *url.URL) error {
	return keyring.Delete(keyringServiceName(registryURL.Host), "")
}

func Token(registryURL *url.URL) (string, error) {
	return keyring.Get(keyringServiceName(registryURL.Host), "")
}

func keyringServiceName(hostname string) string {
	return fmt.Sprintf("pgxman:%s", hostname)
}
