package registry

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pgxman/pgxman/oapi"
)

const (
	defaultHTTPTimeout = 10 * time.Second
)

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

func (c *Client) FindExtension(ctx context.Context, args []string) ([]oapi.SimpleExtension, error) {
	term := strings.Join(args, " ")
	resp, err := c.ListExtensionsWithResponse(ctx, &oapi.ListExtensionsParams{
		Term: &term,
	})
	if err != nil {
		return nil, err
	}

	if resp.JSON500 != nil {
		return nil, fmt.Errorf("server error: %s", resp.JSON500.Message)
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
