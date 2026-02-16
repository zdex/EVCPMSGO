package gatewayclient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) SendCommand(ctx context.Context, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/gateway/commands", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, b, nil
}
