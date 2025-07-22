package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAPIClient(t *testing.T) {
	certSha256 := "1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF"
	client := NewAPIClient(certSha256)

	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}

	if client.client == nil {
		t.Fatal("HTTP client is nil")
	}

	if client.client.Timeout != 30*1000000000 { // 30 seconds in nanoseconds
		t.Errorf("Expected timeout 30s, got %v", client.client.Timeout)
	}
}

func TestGetServerInfo(t *testing.T) {
	// Create a test server
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

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewAPIClient("dummy-cert-sha256")
	serverInfo, err := client.GetServerInfo(server.URL)

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

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewAPIClient("dummy-cert-sha256")
	keys, err := client.ListAccessKeys(server.URL)

	if err != nil {
		t.Fatalf("ListAccessKeys failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
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

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Parse the request body
		var req CreateAccessKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify request fields
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
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewAPIClient("dummy-cert-sha256")
	req := CreateAccessKeyRequest{
		Name:   "Test Key",
		Method: "aes-256-gcm",
		Port:   12345,
	}

	key, err := client.CreateAccessKey(server.URL, req)

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

		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewAPIClient("dummy-cert-sha256")
	err := client.DeleteAccessKey(server.URL, "key123")

	if err != nil {
		t.Fatalf("DeleteAccessKey failed: %v", err)
	}
}

func TestGetTransferMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics/transfer" {
			t.Errorf("Expected path /metrics/transfer, got %s", r.URL.Path)
		}

		response := TransferMetrics{
			BytesTransferredByUserId: map[string]int64{
				"user1": 1024 * 1024, // 1MB
				"user2": 2048 * 1024, // 2MB
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewAPIClient("dummy-cert-sha256")
	metrics, err := client.GetTransferMetrics(server.URL)

	if err != nil {
		t.Fatalf("GetTransferMetrics failed: %v", err)
	}

	if len(metrics.BytesTransferredByUserId) != 2 {
		t.Errorf("Expected 2 users, got %d", len(metrics.BytesTransferredByUserId))
	}

	if metrics.BytesTransferredByUserId["user1"] != 1024*1024 {
		t.Errorf("Expected user1 to have 1MB transferred, got %d", metrics.BytesTransferredByUserId["user1"])
	}
}

func TestRemoveAccessKeyDataLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/access-keys/key123/data-limit" {
			t.Errorf("Expected path /access-keys/key123/data-limit, got %s", r.URL.Path)
		}

		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewAPIClient("dummy-cert-sha256")
	err := client.RemoveAccessKeyDataLimit(server.URL, "key123")

	if err != nil {
		t.Fatalf("RemoveAccessKeyDataLimit failed: %v", err)
	}
}
