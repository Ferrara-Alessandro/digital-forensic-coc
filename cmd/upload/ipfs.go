// Chiamate HTTP al nodo IPFS in locale per caricare file.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

// Carico un file su IPFS e ottengo il cid.
func (c *IPFSClient) AddReader(ctx context.Context, filename string, reader io.Reader) (string, error) {
	pipeReader, pipeWriter := io.Pipe()
	w := multipart.NewWriter(pipeWriter)

	go func() {
		defer func() { _ = pipeWriter.Close() }()
		defer func() { _ = w.Close() }()

		part, err := w.CreateFormFile("file", filename)
		if err != nil {
			_ = pipeWriter.CloseWithError(fmt.Errorf("multipart file: %w", err))
			return
		}
		if _, err := io.Copy(part, reader); err != nil {
			_ = pipeWriter.CloseWithError(fmt.Errorf("multipart copy: %w", err))
			return
		}
	}()

	endpoint, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("ipfs url: %w", err)
	}
	endpoint.Path = path.Join(endpoint.Path, "/api/v0/add")
	q := endpoint.Query()
	q.Set("pin", "false")
	// Nodo locale: non attendere la rete IPFS pubblica.
	q.Set("offline", "true")
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), pipeReader)
	if err != nil {
		return "", fmt.Errorf("request add: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ipfs add: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ipfs read resp: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ipfs add status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Hash string `json:"Hash"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("ipfs parse resp: %w", err)
	}
	if parsed.Hash == "" {
		return "", fmt.Errorf("ipfs cid vuoto")
	}
	return parsed.Hash, nil
}
