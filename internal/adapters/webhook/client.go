package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/muratdemir0/gopulse-messages/internal/adapters/ohttp"
	"net/http"
)

type Client struct {
	Host       string
	httpClient *ohttp.Client
}

type Response struct {
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}

type Request struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

func NewClient(host string, httpClient *ohttp.Client) *Client {
	return &Client{
		Host:       host,
		httpClient: httpClient,
	}
}

func (c *Client) Send(ctx context.Context, message Request, path string) (*Response, error) {
	payload, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Host+path, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}
