package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pgxman/pgxman/oapi"
)

const (
	defaultHTTPTimeout = 10 * time.Second
)

var (
	ErrExtensionNotFound = errors.New("extension not found")
)

func NewClient(baseURL, token string) (Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	user := u.User
	c, err := oapi.NewClientWithResponses(
		baseURL,
		oapi.WithHTTPClient(
			&http.Client{
				Timeout: defaultHTTPTimeout,
			},
		),
		oapi.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			// use jwt auth when basic auth creds are not in the base URL
			if user == nil && token != "" {
				req.Header.Add("Authorization", fmt.Sprintf("bearer %s", token))
			}

			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	return &client{
		ClientWithResponsesInterface: c,
	}, nil
}

type Client interface {
	GetExtension(ctx context.Context, name string) (*oapi.Extension, error)
	FindExtension(ctx context.Context, args []string) ([]oapi.SimpleExtension, error)
	PublishExtension(ctx context.Context, ext oapi.PublishExtension) error
	GetVersion(ctx context.Context, name, version string) (*oapi.Extension, error)
	GetUser(ctx context.Context) (*oapi.User, error)
}

type client struct {
	oapi.ClientWithResponsesInterface
}

func (c *client) GetUser(ctx context.Context) (*oapi.User, error) {
	resp, err := c.ClientWithResponsesInterface.GetUserWithResponse(ctx)
	if err != nil {
		return nil, err
	}

	var errMsg string
	if resp.JSON401 != nil {
		errMsg = resp.JSON401.Message
	} else if resp.HTTPResponse.StatusCode >= 300 {
		errMsg = strings.TrimSpace(string(resp.Body))
	}
	if errMsg != "" {
		return nil, fmt.Errorf("error getting user: %s", errMsg)
	}

	return resp.JSON200, nil
}

func (c *client) GetExtension(ctx context.Context, name string) (*oapi.Extension, error) {
	resp, err := c.ClientWithResponsesInterface.FindExtensionWithResponse(ctx, name)
	if err != nil {
		return nil, err
	}

	if resp.JSON404 != nil {
		return nil, ErrExtensionNotFound
	}

	var errMsg string
	if resp.JSON400 != nil {
		errMsg = resp.JSON400.Message
	}
	if resp.JSON500 != nil {
		errMsg = resp.JSON500.Message
	}
	if errMsg != "" {
		return nil, fmt.Errorf("error getting %s: %s", name, errMsg)
	}

	return resp.JSON200, nil
}

func (c *client) FindExtension(ctx context.Context, args []string) ([]oapi.SimpleExtension, error) {
	term := strings.Join(args, " ")
	resp, err := c.ListExtensionsWithResponse(ctx, &oapi.ListExtensionsParams{
		Term: &term,
	})
	if err != nil {
		return nil, err
	}

	if resp.JSON500 != nil {
		return nil, fmt.Errorf("error finding extension: %s", resp.JSON500.Message)
	}

	return resp.JSON200.Extensions, nil
}

func (c *client) PublishExtension(ctx context.Context, ext oapi.PublishExtension) error {
	resp, err := c.PublishExtensionWithResponse(
		ctx,
		ext,
	)
	if err != nil {
		return err
	}

	var errMsg string
	if resp.JSON400 != nil {
		errMsg = resp.JSON400.Message
	} else if resp.JSON500 != nil {
		errMsg = resp.JSON500.Message
	} else if resp.HTTPResponse.StatusCode == http.StatusUnauthorized {
		errMsg = strings.TrimSpace(string(resp.Body))
	}
	if errMsg != "" {
		return fmt.Errorf("error publishing %s: %s", ext.Name, errMsg)
	}

	return nil
}

func (c *client) GetVersion(ctx context.Context, name, version string) (*oapi.Extension, error) {
	resp, err := c.ClientWithResponsesInterface.FindVersionWithResponse(ctx, name, version)
	if err != nil {
		return nil, err
	}

	if resp.JSON404 != nil {
		return nil, ErrExtensionNotFound
	}

	var errMsg string
	if resp.JSON400 != nil {
		errMsg = resp.JSON400.Message
	}
	if resp.JSON500 != nil {
		errMsg = resp.JSON500.Message
	}
	if errMsg != "" {
		return nil, fmt.Errorf("error getting extension version %s: %s", name, errMsg)
	}

	return resp.JSON200, nil
}
