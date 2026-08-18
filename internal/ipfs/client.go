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

// Client는 IPFS HTTP API와 통신하는 클라이언트다.
// apiURL: IPFS 노드 주소 (기본값: "http://localhost:5001")
type Client struct {
	apiURL          string
	http            *http.Client
	maxDownloadSize int64
}

const MaxDownloadBytes int64 = 64 << 20

// NewClient는 IPFS 클라이언트를 생성한다.
func NewClient(apiURL string) *Client {
	return &Client{
		apiURL:          strings.TrimRight(apiURL, "/"),
		maxDownloadSize: MaxDownloadBytes,
		http: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

// Upload는 데이터를 IPFS에 업로드하고 CID 문자열을 반환한다.
// multipart/form-data 형식으로 POST /api/v0/add를 호출한다.
// 반환된 CID는 이후 Download 호출이나 온체인 기록에 사용된다.
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

	// 응답 JSON에서 CID(Hash 필드) 추출
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

// Download는 CID로 IPFS에서 데이터를 다운로드한다.
// POST /api/v0/cat?arg=<cid>를 호출해 원본 바이트를 반환한다.
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
