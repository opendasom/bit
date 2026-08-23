// Package ipfs provides a minimal HTTP client for uploading and downloading
// content-addressed data to/from a Kubo IPFS node, used for storing commit
// diffs, manifests, and repo metadata off-chain.
package ipfs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to a Kubo IPFS node over its HTTP API.
// apiURL is the IPFS node address (default: "http://localhost:5001").
type Client struct {
	apiURL          string
	http            *http.Client
	maxDownloadSize int64
}

const MaxDownloadBytes int64 = 64 << 20

// NewClient creates an IPFS client bound to the node at apiURL.
func NewClient(apiURL string) *Client {
	return &Client{
		apiURL:          strings.TrimRight(apiURL, "/"),
		maxDownloadSize: MaxDownloadBytes,
		http: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

// Upload adds data to IPFS and returns its CID string, by POSTing
// multipart/form-data to /api/v0/add. The returned CID is later used for
// Download calls and for on-chain records.
func (c *Client) Upload(data []byte) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "data")
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	resp, err := c.http.Post(
		c.apiURL+"/api/v0/add",
		writer.FormDataContentType(),
		body,
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("ipfs add failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	responseData, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024+1))
	if err != nil {
		return "", err
	}
	if len(responseData) > 64*1024 {
		return "", fmt.Errorf("ipfs add response exceeds maximum size of %d bytes", 64*1024)
	}

	// Extract the CID (the Hash field) from the response JSON.
	var result struct {
		Hash string `json:"Hash"`
	}
	if err := json.Unmarshal(responseData, &result); err != nil {
		return "", err
	}
	if result.Hash == "" {
		return "", fmt.Errorf("ipfs add response did not include Hash")
	}
	return result.Hash, nil
}

// Download fetches data from IPFS by CID, by POSTing to
// /api/v0/cat?arg=<cid>, and returns the raw bytes.
func (c *Client) Download(cid string) ([]byte, error) {
	resp, err := c.http.Post(
		fmt.Sprintf("%s/api/v0/cat?arg=%s", c.apiURL, url.QueryEscape(cid)),
		"",
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("ipfs cat failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, c.maxDownloadSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > c.maxDownloadSize {
		return nil, fmt.Errorf("ipfs object exceeds maximum size of %d bytes", c.maxDownloadSize)
	}
	return data, nil
}
