// Scarico contenuti da IPFS con chiamate HTTP.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type IPFSClient struct {
	baseURL string
	client  *http.Client
}

func NewIPFSClient(baseURL string, timeout time.Duration) *IPFSClient {
	return &IPFSClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

// Dato un cid, scarico i byte del file cifrato.
func (c *IPFSClient) CatBytes(ctx context.Context, cid string) ([]byte, error) {
	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("ipfs url: %w", err)
	}
	endpoint.Path = path.Join(endpoint.Path, "/api/v0/cat")
	q := endpoint.Query()
	q.Set("arg", cid)
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("request cat: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ipfs cat: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ipfs read resp: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ipfs cat status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}
