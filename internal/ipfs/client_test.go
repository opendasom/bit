package ipfs

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestUploadReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	_, err := NewClient(server.URL).Upload([]byte("data"))
	if err == nil {
		t.Fatal("expected upload error")
	}
}

func TestDownloadRejectsOversizedObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(bytes.Repeat([]byte{'x'}, 33))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.maxDownloadSize = 32
	_, err := client.Download("oversized")
	if err == nil {
		t.Fatal("expected oversized download error")
	}
}

func TestUploadRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte{'x'}, 64*1024+1))
	}))
	defer server.Close()

	_, err := NewClient(server.URL).Upload([]byte("data"))
	if err == nil {
		t.Fatal("expected oversized upload response error")
	}
}

func TestLiveKuboUploadDownload(t *testing.T) {
	apiURL := os.Getenv("BIT_IPFS_API")
	if apiURL == "" {
		t.Skip("BIT_IPFS_API is not set")
	}

	client := NewClient(apiURL)
	payload := []byte("bit live ipfs verification\n")
	cid, err := client.Upload(payload)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Download(cid)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("downloaded payload mismatch: got %q, want %q", got, payload)
	}
}
