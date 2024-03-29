package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"golang.org/x/oauth2"
)

type Screen int

const (
	LoginScreen Screen = iota
	SignupScreen
)

type FlowParams struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
	Audience     string
	Endpoint     string
	Screen       Screen
}

func (p FlowParams) Validate() error {
	var result error

	if p.ClientID == "" {
		result = errors.Join(result, errors.New("client ID is required"))
	}

	_, err := url.ParseRequestURI(p.Endpoint)
	if err != nil {
		result = errors.Join(result, fmt.Errorf("invalid endpoint %s: %w", p.Endpoint, err))
	}

	_, err = url.ParseRequestURI(p.Audience)
	if err != nil {
		result = errors.Join(result, fmt.Errorf("invalid endpoint %s: %w", p.Audience, err))
	}

	return result
}

func InitFlow(params FlowParams) (*Flow, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	server, err := bindLocalServer()
	if err != nil {
		return nil, err
	}

	// wait for the server to start
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		_ = server.Serve()
	}()
	wg.Wait()

	return &Flow{
		server:   server,
		params:   params,
		verifier: oauth2.GenerateVerifier(),
	}, nil
}

type Flow struct {
	server   *localServer
	params   FlowParams
	verifier string
}

func (f *Flow) Done() error {
	return f.server.Close()
}

func (f *Flow) BrowserURL() string {
	var screenHint oauth2.AuthCodeOption
	switch f.params.Screen {
	case SignupScreen:
		screenHint = oauth2.SetAuthURLParam("screen_hint", "signup")
	default:
		screenHint = oauth2.SetAuthURLParam("prompt", "login")
	}

	return f.conf().AuthCodeURL(
		"",
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(f.verifier),
		oauth2.SetAuthURLParam("audience", f.params.Audience),
		screenHint,
	)
}

func (f *Flow) WaitForToken(ctx context.Context) (string, error) {
	code, err := f.server.WaitForCode(ctx)
	if err != nil {
		return "", err
	}

	if code.Code == "" {
		return "", errors.New("no oauth code received")
	}

	tok, err := f.conf().Exchange(ctx, code.Code, oauth2.VerifierOption(f.verifier))
	if err != nil {
		return "", err
	}

	return tok.AccessToken, nil
}

func (c *Flow) conf() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.params.ClientID,
		ClientSecret: c.params.ClientSecret,
		Scopes:       c.params.Scopes,
		RedirectURL:  c.server.URL(),
		Endpoint: oauth2.Endpoint{
			AuthURL:       fmt.Sprintf("%s/authorize", c.params.Endpoint),
			DeviceAuthURL: fmt.Sprintf("%s/oauth/device/code", c.params.Endpoint),
			TokenURL:      fmt.Sprintf("%s/oauth/token", c.params.Endpoint),
		},
	}
}
