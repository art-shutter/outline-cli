package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client handles HTTP requests to an Outline VPN management API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates an Outline API client for serverURL pinned to the given certificate SHA-256 fingerprint.
func NewClient(serverURL, certSha256 string) *Client {
	expectedHash := strings.ToUpper(strings.TrimSpace(certSha256))

	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(serverURL), "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					// Outline uses self-signed certificates verified against a pinned SHA-256 hash.
					InsecureSkipVerify: true, //nolint:gosec
					VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
						if len(rawCerts) == 0 {
							return errors.New("server presented no TLS certificates")
						}

						hash := sha256.Sum256(rawCerts[0])
						gotHash := strings.ToUpper(hex.EncodeToString(hash[:]))
						if gotHash != expectedHash {
							return fmt.Errorf("certificate SHA256 mismatch: expected %s, got %s", expectedHash, gotHash)
						}

						return nil
					},
				},
			},
		},
	}
}

type DataLimit struct {
	Bytes int64 `json:"bytes"`
}

type OutlineServer struct {
	Name                  string     `json:"name"`
	ServerID              string     `json:"serverId"`
	MetricsEnabled        bool       `json:"metricsEnabled"`
	CreatedTimestampMs    int64      `json:"createdTimestampMs"`
	Version               string     `json:"version"`
	PortForNewAccessKeys  int        `json:"portForNewAccessKeys"`
	HostnameForAccessKeys string     `json:"hostnameForAccessKeys"`
	AccessKeyDataLimit    *DataLimit `json:"accessKeyDataLimit,omitempty"`
}

type AccessKey struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Password  string     `json:"password"`
	Port      int        `json:"port"`
	Method    string     `json:"method"`
	AccessURL string     `json:"accessUrl"`
	DataLimit *DataLimit `json:"dataLimit,omitempty"`
}

type CreateAccessKeyRequest struct {
	Name     string     `json:"name,omitempty"`
	Method   string     `json:"method,omitempty"`
	Password string     `json:"password,omitempty"`
	Port     int        `json:"port,omitempty"`
	Limit    *DataLimit `json:"limit,omitempty"`
}

type AccessKeysResponse struct {
	AccessKeys []AccessKey `json:"accessKeys"`
}

type TransferMetrics struct {
	BytesTransferredByUserID map[string]int64 `json:"bytesTransferredByUserId"`
}

type renameAccessKeyRequest struct {
	Name string `json:"name"`
}

type setDataLimitRequest struct {
	Limit DataLimit `json:"limit"`
}

func (c *Client) GetServerInfo(ctx context.Context) (*OutlineServer, error) {
	var server OutlineServer
	if err := c.do(ctx, http.MethodGet, "/server", nil, &server, http.StatusOK); err != nil {
		return nil, fmt.Errorf("get server info: %w", err)
	}
	return &server, nil
}

func (c *Client) ListAccessKeys(ctx context.Context) ([]AccessKey, error) {
	var resp AccessKeysResponse
	if err := c.do(ctx, http.MethodGet, "/access-keys", nil, &resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("list access keys: %w", err)
	}
	return resp.AccessKeys, nil
}

func (c *Client) CreateAccessKey(ctx context.Context, req CreateAccessKeyRequest) (*AccessKey, error) {
	var key AccessKey
	if err := c.do(ctx, http.MethodPost, "/access-keys", req, &key, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("create access key: %w", err)
	}
	return &key, nil
}

func (c *Client) DeleteAccessKey(ctx context.Context, keyID string) error {
	path := "/access-keys/" + url.PathEscape(keyID)
	if err := c.do(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent); err != nil {
		return fmt.Errorf("delete access key %q: %w", keyID, err)
	}
	return nil
}

func (c *Client) GetTransferMetrics(ctx context.Context) (*TransferMetrics, error) {
	var metrics TransferMetrics
	if err := c.do(ctx, http.MethodGet, "/metrics/transfer", nil, &metrics, http.StatusOK); err != nil {
		return nil, fmt.Errorf("get transfer metrics: %w", err)
	}
	return &metrics, nil
}

func (c *Client) RenameAccessKey(ctx context.Context, keyID, newName string) error {
	path := "/access-keys/" + url.PathEscape(keyID) + "/name"
	req := renameAccessKeyRequest{Name: newName}
	if err := c.do(ctx, http.MethodPut, path, req, nil, http.StatusNoContent); err != nil {
		return fmt.Errorf("rename access key %q: %w", keyID, err)
	}
	return nil
}

func (c *Client) SetAccessKeyDataLimit(ctx context.Context, keyID string, limit DataLimit) error {
	path := "/access-keys/" + url.PathEscape(keyID) + "/data-limit"
	req := setDataLimitRequest{Limit: limit}
	if err := c.do(ctx, http.MethodPut, path, req, nil, http.StatusNoContent); err != nil {
		return fmt.Errorf("set data limit for key %q: %w", keyID, err)
	}
	return nil
}

func (c *Client) RemoveAccessKeyDataLimit(ctx context.Context, keyID string) error {
	path := "/access-keys/" + url.PathEscape(keyID) + "/data-limit"
	if err := c.do(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent); err != nil {
		return fmt.Errorf("remove data limit for key %q: %w", keyID, err)
	}
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, reqBody, respDest any, expectedStatus int) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != expectedStatus {
		body, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			return fmt.Errorf("unexpected HTTP status %d (expected %d)", resp.StatusCode, expectedStatus)
		}
		return fmt.Errorf("unexpected HTTP status %d (expected %d): %s", resp.StatusCode, expectedStatus, msg)
	}

	if respDest != nil {
		if err := json.NewDecoder(resp.Body).Decode(respDest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
