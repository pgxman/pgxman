package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

func NewClient(baseURL string) (Client, error) {
	c, err := oapi.NewClientWithResponses(
		baseURL,
		oapi.WithHTTPClient(
			&http.Client{
				Timeout: defaultHTTPTimeout,
			},
		),
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
}

type client struct {
	oapi.ClientWithResponsesInterface
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
