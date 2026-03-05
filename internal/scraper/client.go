package scraper

import (
	"context"
	"net/http"
	"time"
)

type Client struct {
	HTTP *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{HTTP: httpClient}
}

func (c *Client) Fetch(ctx context.Context, url string) (int, []byte, error) {
	return Fetch(ctx, c.HTTP, url)
}

func (c *Client) ParseDays(body []byte) ([]Day, error) {
	return ParseDays(body)
}
