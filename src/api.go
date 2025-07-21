package main

import (
	"bytes"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/goccy/go-json"
)

// closeResponseBody safely closes the response body and logs any error
func closeResponseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", "error", err)
		}
	}
}

// APIClient handles HTTP requests to Outline servers
type APIClient struct {
	client *http.Client
}

// NewAPIClient creates a new API client
func NewAPIClient() *APIClient {
	return &APIClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				// we need to ignore tls as outline servers use self-signed ones
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
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
	BytesTransferredByUserId map[string]int64 `json:"bytesTransferredByUserId"`
}

func (api *APIClient) GetServerInfo(serverURL string) (*OutlineServer, error) {
	resp, err := api.client.Get(serverURL + "/server")
	if err != nil {
		slog.Error("failed to get server info", "error", err)
		return nil, err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("server returned status", "status", resp.StatusCode, "body", string(body))
		return nil, err
	}

	var server OutlineServer
	if err := json.NewDecoder(resp.Body).Decode(&server); err != nil {
		slog.Error("failed to decode server response", "error", err)
		return nil, err
	}

	return &server, nil
}

func (api *APIClient) ListAccessKeys(serverURL string) ([]AccessKey, error) {
	resp, err := api.client.Get(serverURL + "/access-keys")
	if err != nil {
		slog.Error("failed to list access keys", "error", err)
		return nil, err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("server returned status", "status", resp.StatusCode, "body", string(body))
		return nil, err
	}

	var response AccessKeysResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		slog.Error("failed to decode access keys response", "error", err)
		return nil, err
	}

	return response.AccessKeys, nil
}

func (api *APIClient) CreateAccessKey(serverURL string, req CreateAccessKeyRequest) (*AccessKey, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		slog.Error("failed to marshal request", "error", err)
		return nil, err
	}

	resp, err := api.client.Post(serverURL+"/access-keys", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("failed to create access key", "error", err)
		return nil, err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("server returned status", "status", resp.StatusCode, "body", string(body))
		return nil, err
	}

	var accessKey AccessKey
	if err := json.NewDecoder(resp.Body).Decode(&accessKey); err != nil {
		slog.Error("failed to decode access key response", "error", err)
		return nil, err
	}

	return &accessKey, nil
}

func (api *APIClient) DeleteAccessKey(serverURL, keyID string) error {
	req, err := http.NewRequest("DELETE", serverURL+"/access-keys/"+url.PathEscape(keyID), nil)
	if err != nil {
		slog.Error("failed to create delete request", "error", err)
		return err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		slog.Error("failed to delete access key", "error", err)
		return err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("server returned status", "status", resp.StatusCode, "body", string(body))
		return err
	}

	return nil
}

func (api *APIClient) GetTransferMetrics(serverURL string) (*TransferMetrics, error) {
	resp, err := api.client.Get(serverURL + "/metrics/transfer")
	if err != nil {
		slog.Error("failed to get transfer metrics", "error", err)
		return nil, err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("server returned status", "status", resp.StatusCode, "body", string(body))
		return nil, err
	}

	var metrics TransferMetrics
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		slog.Error("failed to decode metrics response", "error", err)
		return nil, err
	}

	return &metrics, nil
}

func (api *APIClient) RenameAccessKey(serverURL, keyID, newName string) error {
	reqData := map[string]string{"name": newName}
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		slog.Error("failed to marshal request", "error", err)
		return err
	}

	req, err := http.NewRequest("PUT", serverURL+"/access-keys/"+url.PathEscape(keyID)+"/name", bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("failed to create rename request", "error", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		slog.Error("failed to rename access key", "error", err)
		return err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("server returned status", "status", resp.StatusCode, "body", string(body))
		return err
	}

	return nil
}

func (api *APIClient) SetAccessKeyDataLimit(serverURL, keyID string, limit DataLimit) error {
	reqData := map[string]DataLimit{"limit": limit}
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		slog.Error("failed to marshal request", "error", err)
		return err
	}

	req, err := http.NewRequest("PUT", serverURL+"/access-keys/"+url.PathEscape(keyID)+"/data-limit", bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("failed to create data limit request", "error", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		slog.Error("failed to set access key data limit", "error", err)
		return err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("server returned status", "status", resp.StatusCode, "body", string(body))
		return err
	}

	return nil
}

func (api *APIClient) RemoveAccessKeyDataLimit(serverURL, keyID string) error {
	req, err := http.NewRequest("DELETE", serverURL+"/access-keys/"+url.PathEscape(keyID)+"/data-limit", nil)
	if err != nil {
		slog.Error("failed to create remove data limit request", "error", err)
		return err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		slog.Error("failed to remove access key data limit", "error", err)
		return err
	}
	defer closeResponseBody(resp)

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("server returned status", "status", resp.StatusCode, "body", string(body))
		return err
	}

	return nil
}
