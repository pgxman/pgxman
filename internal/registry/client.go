package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/oapi"
)

const (
	defaultHTTPTimeout = 10 * time.Second
)

var (
	ErrExtensionNotFound = errors.New("extension not found")
)

type Extension struct {
	oapi.Extension
}

func (e *Extension) GetPlatform(p pgxman.Platform) (*oapi.Platform, error) {
	for _, platform := range e.Platforms {
		if string(platform.Os) == string(p) {
			return &platform, nil
		}
	}

	return nil, fmt.Errorf("platform %q not found", p)
}

func NewClient(baseURL string) (*Client, error) {
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

	return &Client{
		ClientWithResponsesInterface: c,
	}, nil
}

type Client struct {
	oapi.ClientWithResponsesInterface
}

func (c *Client) GetExtension(ctx context.Context, name string) (*Extension, error) {
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

	return &Extension{Extension: *resp.JSON200}, nil
}

func (c *Client) FindExtension(ctx context.Context, args []string) ([]oapi.SimpleExtension, error) {
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

func (c *Client) PublishExtension(ctx context.Context, ext oapi.PublishExtension) error {
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
	}
	if resp.JSON500 != nil {
		errMsg = resp.JSON500.Message
	}
	if errMsg != "" {
		return fmt.Errorf("error publishing %s: %s", ext.Name, errMsg)
	}

	return nil
}

func (c *Client) GetVersion(ctx context.Context, name, version string) (*Extension, error) {
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

	return &Extension{Extension: *resp.JSON200}, nil
}
