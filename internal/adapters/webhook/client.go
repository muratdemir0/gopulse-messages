package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"github.com/muratdemir0/gopulse-messages/internal/adapters/ohttp"
)

type Client struct {
	Host       string
	httpClient *ohttp.Client
}

type Response struct {
	Message      string `json:"message"`
	MessageID    string `json:"messageId"`
	RetryAttempt int    `json:"-"`
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

	fullUrl := fmt.Sprintf("%s/%s", c.Host, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullUrl, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d for %s", resp.StatusCode, fullUrl)
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	response.RetryAttempt, err = strconv.Atoi(resp.Header.Get(ohttp.HeaderRetryAttempt))
	if err != nil {
		return nil, err
	}

	return &response, nil
}
