package apm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const elasticAPIVersion = "2023-10-31"

type Config struct {
	BaseURL    string
	APIKey     string
	Headers    map[string]string
	HTTPClient *http.Client
}

type Client struct {
	baseURL    string
	apiKey     string
	headers    map[string]string
	httpClient *http.Client
}

func New(cfg Config) *Client {
	base := strings.TrimRight(cfg.BaseURL, "/")
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		baseURL:    base,
		apiKey:     cfg.APIKey,
		headers:    cfg.Headers,
		httpClient: hc,
	}
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	rawURL := c.baseURL + path
	if len(query) > 0 {
		rawURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("apm: build request: %w", err)
	}
	req.Header.Set("Elastic-Api-Version", elasticAPIVersion)
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, query url.Values, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("apm: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	rawURL := c.baseURL + path
	if len(query) > 0 {
		rawURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bodyReader)
	if err != nil {
		return fmt.Errorf("apm: build request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Elastic-Api-Version", elasticAPIVersion)
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	req.Header.Set("Authorization", "ApiKey "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("kbn-xsrf", "true")
	req.Header.Set("x-elastic-internal-origin", "Kibana")

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("apm: http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return c.handleError(resp)
	}

	if out == nil {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("apm: read body: %w", err)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("apm: decode: %w", err)
	}
	return nil
}

func (c *Client) handleError(resp *http.Response) error {
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusForbidden:
		return ErrForbidden
	}
	body, _ := io.ReadAll(resp.Body)
	return &APIError{StatusCode: resp.StatusCode, Message: string(body)}
}
