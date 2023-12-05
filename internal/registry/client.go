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
