package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	certSha256 := "1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF"
	client := NewClient("https://example.com/secret/", certSha256)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.baseURL != "https://example.com/secret" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://example.com/secret")
	}
	if client.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", client.httpClient.Timeout)
	}
}

func TestGetServerInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server" {
			t.Errorf("Expected path /server, got %s", r.URL.Path)
		}

		response := OutlineServer{
			Name:                  "Test Server",
			ServerID:              "test-server-id",
			MetricsEnabled:        true,
			CreatedTimestampMs:    1640995200000,
			Version:               "1.0.0",
			PortForNewAccessKeys:  12345,
			HostnameForAccessKeys: "test.example.com",
		}

		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "dummy-cert-sha256")
	serverInfo, err := client.GetServerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetServerInfo failed: %v", err)
	}

	if serverInfo.Name != "Test Server" {
		t.Errorf("Expected server name 'Test Server', got %s", serverInfo.Name)
	}
	if serverInfo.ServerID != "test-server-id" {
		t.Errorf("Expected server ID 'test-server-id', got %s", serverInfo.ServerID)
	}
	if !serverInfo.MetricsEnabled {
		t.Error("Expected metrics to be enabled")
	}
}

func TestGetServerInfo_ErrorOnNon200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient(server.URL, "dummy-cert-sha256")
	_, err := client.GetServerInfo(context.Background())
	if err == nil {
		t.Fatal("expected error on HTTP 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error to mention status 403, got: %v", err)
	}
}

func TestTLSCertPinning(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(OutlineServer{Name: "Pinned TLS Server"})
	}))
	defer server.Close()

	cert := server.Certificate()
	sum := sha256.Sum256(cert.Raw)
	validHash := hex.EncodeToString(sum[:])

	t.Run("matching fingerprint succeeds", func(t *testing.T) {
		client := NewClient(server.URL, validHash)
		info, err := client.GetServerInfo(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Name != "Pinned TLS Server" {
			t.Errorf("Name = %q, want %q", info.Name, "Pinned TLS Server")
		}
	})

	t.Run("mismatched fingerprint fails", func(t *testing.T) {
		client := NewClient(server.URL, "0000000000000000000000000000000000000000000000000000000000000000")
		_, err := client.GetServerInfo(context.Background())
		if err == nil {
			t.Fatal("expected certificate mismatch error, got nil")
		}
		if !strings.Contains(err.Error(), "certificate SHA256 mismatch") {
			t.Errorf("expected certificate mismatch error, got: %v", err)
		}
	})
}

func TestListAccessKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/access-keys" {
			t.Errorf("Expected path /access-keys, got %s", r.URL.Path)
		}

		response := AccessKeysResponse{
			AccessKeys: []AccessKey{
				{
					ID:        "key1",
					Name:      "Test Key 1",
					Password:  "password1",
					Port:      12345,
					Method:    "aes-256-gcm",
					AccessURL: "ss://test1",
				},
				{
					ID:        "key2",
					Name:      "Test Key 2",
					Password:  "password2",
					Port:      12346,
					Method:    "chacha20-poly1305",
					AccessURL: "ss://test2",
				},
			},
		}

		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "dummy-cert-sha256")
	keys, err := client.ListAccessKeys(context.Background())
	if err != nil {
		t.Fatalf("ListAccessKeys failed: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("Expected 2 keys, got %d", len(keys))
	}
	if keys[0].ID != "key1" {
		t.Errorf("Expected first key ID 'key1', got %s", keys[0].ID)
	}
	if keys[1].Name != "Test Key 2" {
		t.Errorf("Expected second key name 'Test Key 2', got %s", keys[1].Name)
	}
}

func TestCreateAccessKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/access-keys" {
			t.Errorf("Expected path /access-keys, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		var req CreateAccessKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if req.Name != "Test Key" {
			t.Errorf("Expected name 'Test Key', got %s", req.Name)
		}
		if req.Method != "aes-256-gcm" {
			t.Errorf("Expected method 'aes-256-gcm', got %s", req.Method)
		}

		response := AccessKey{
			ID:        "new-key-id",
			Name:      req.Name,
			Password:  "generated-password",
			Port:      req.Port,
			Method:    req.Method,
			AccessURL: "ss://new-key",
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "dummy-cert-sha256")
	key, err := client.CreateAccessKey(context.Background(), CreateAccessKeyRequest{
		Name:   "Test Key",
		Method: "aes-256-gcm",
		Port:   12345,
	})
	if err != nil {
		t.Fatalf("CreateAccessKey failed: %v", err)
	}

	if key.ID != "new-key-id" {
		t.Errorf("Expected key ID 'new-key-id', got %s", key.ID)
	}
	if key.Name != "Test Key" {
		t.Errorf("Expected key name 'Test Key', got %s", key.Name)
	}
}

func TestDeleteAccessKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/access-keys/key123" {
			t.Errorf("Expected path /access-keys/key123, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "dummy-cert-sha256")
	if err := client.DeleteAccessKey(context.Background(), "key123"); err != nil {
		t.Fatalf("DeleteAccessKey failed: %v", err)
	}
}

func TestRenameAndSetDataLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/access-keys/key123/name":
			if r.Method != http.MethodPut {
				t.Errorf("Expected PUT method, got %s", r.Method)
			}
			var body renameAccessKeyRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode rename body: %v", err)
			}
			if body.Name != "Renamed Key" {
				t.Errorf("Name = %q, want %q", body.Name, "Renamed Key")
			}
			w.WriteHeader(http.StatusNoContent)
		case "/access-keys/key123/data-limit":
			if r.Method != http.MethodPut {
				t.Errorf("Expected PUT method, got %s", r.Method)
			}
			var body setDataLimitRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode limit body: %v", err)
			}
			if body.Limit.Bytes != 1000000000 {
				t.Errorf("Limit.Bytes = %d, want %d", body.Limit.Bytes, 1000000000)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "dummy-cert-sha256")
	if err := client.RenameAccessKey(context.Background(), "key123", "Renamed Key"); err != nil {
		t.Fatalf("RenameAccessKey failed: %v", err)
	}
	if err := client.SetAccessKeyDataLimit(context.Background(), "key123", DataLimit{Bytes: 1000000000}); err != nil {
		t.Fatalf("SetAccessKeyDataLimit failed: %v", err)
	}
}

func TestGetTransferMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics/transfer" {
			t.Errorf("Expected path /metrics/transfer, got %s", r.URL.Path)
		}

		response := TransferMetrics{
			BytesTransferredByUserID: map[string]int64{
				"user1": 1024 * 1024,
				"user2": 2048 * 1024,
			},
		}

		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "dummy-cert-sha256")
	metrics, err := client.GetTransferMetrics(context.Background())
	if err != nil {
		t.Fatalf("GetTransferMetrics failed: %v", err)
	}

	if len(metrics.BytesTransferredByUserID) != 2 {
		t.Errorf("Expected 2 users, got %d", len(metrics.BytesTransferredByUserID))
	}
	if metrics.BytesTransferredByUserID["user1"] != 1024*1024 {
		t.Errorf("Expected user1 to have 1MB transferred, got %d", metrics.BytesTransferredByUserID["user1"])
	}
}

func TestRemoveAccessKeyDataLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/access-keys/key123/data-limit" {
			t.Errorf("Expected path /access-keys/key123/data-limit, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "dummy-cert-sha256")
	if err := client.RemoveAccessKeyDataLimit(context.Background(), "key123"); err != nil {
		t.Fatalf("RemoveAccessKeyDataLimit failed: %v", err)
	}
}
