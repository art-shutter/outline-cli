package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/art-shutter/outline-cli/internal/api"
	"github.com/art-shutter/outline-cli/internal/config"
)

func TestExecuteEndToEnd(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/server":
			_ = json.NewEncoder(w).Encode(api.OutlineServer{
				Name:                  "Test Outline",
				ServerID:              "srv-1",
				Version:               "1.9.0",
				MetricsEnabled:        true,
				PortForNewAccessKeys:  12345,
				HostnameForAccessKeys: "vpn.example.com",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/access-keys":
			_ = json.NewEncoder(w).Encode(api.AccessKeysResponse{
				AccessKeys: []api.AccessKey{
					{
						ID:        "1",
						Name:      "Laptop",
						Password:  "secret",
						Port:      12345,
						Method:    "aes-192-gcm",
						AccessURL: "ss://laptop",
						DataLimit: &api.DataLimit{Bytes: 1000000000},
					},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/access-keys":
			var req api.CreateAccessKeyRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(api.AccessKey{
				ID:        "2",
				Name:      req.Name,
				Password:  "pw2",
				Port:      req.Port,
				Method:    req.Method,
				AccessURL: "ss://phone",
				DataLimit: req.Limit,
			})
		case r.Method == http.MethodPut && r.URL.Path == "/access-keys/1/name":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && r.URL.Path == "/access-keys/1/data-limit":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/access-keys/1/data-limit":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/access-keys/1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/metrics/transfer":
			_ = json.NewEncoder(w).Encode(api.TransferMetrics{
				BytesTransferredByUserID: map[string]int64{"1": 5000000},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	sum := sha256.Sum256(server.Certificate().Raw)
	certHash := hex.EncodeToString(sum[:])

	store, err := config.Load(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("Load store: %v", err)
	}

	ctx := context.Background()

	// 1. Add server
	var out bytes.Buffer
	err = execute(ctx, &Args{
		Servers: &ServersCmd{
			Add: &AddCmd{
				Name:       "myserver",
				URL:        ServerURL(server.URL),
				CertSha256: CertSHA256(certHash),
			},
		},
	}, store, &out)
	if err != nil {
		t.Fatalf("servers add: %v", err)
	}

	// 2. Get server info
	out.Reset()
	err = execute(ctx, &Args{
		Servers: &ServersCmd{
			Get: &GetCmd{Name: "myserver"},
		},
	}, store, &out)
	if err != nil {
		t.Fatalf("servers get: %v", err)
	}
	if !strings.Contains(out.String(), "Test Outline") {
		t.Errorf("expected server info in output, got:\n%s", out.String())
	}

	// 3. Create key
	out.Reset()
	err = execute(ctx, &Args{
		Keys: &KeysCmd{
			Create: &CreateKeyCmd{
				ServerName: "myserver",
				Name:       "Phone",
				Method:     MethodAES192GCM,
				Port:       Port(12345),
				DataLimit:  DataSize(1000000000),
			},
		},
	}, store, &out)
	if err != nil {
		t.Fatalf("keys create: %v", err)
	}
	if !strings.Contains(out.String(), "Access key created successfully!") {
		t.Errorf("unexpected create key output:\n%s", out.String())
	}

	// 4. List keys
	out.Reset()
	err = execute(ctx, &Args{
		Keys: &KeysCmd{
			List: &ListKeysCmd{ServerName: "myserver"},
		},
	}, store, &out)
	if err != nil {
		t.Fatalf("keys list: %v", err)
	}
	if !strings.Contains(out.String(), "Laptop") {
		t.Errorf("expected Laptop in keys list, got:\n%s", out.String())
	}

	// 5. Edit key by name
	out.Reset()
	err = execute(ctx, &Args{
		Keys: &KeysCmd{
			Edit: &EditKeyCmd{
				ServerName: "myserver",
				KeyName:    "Laptop",
				NewName:    "Work Laptop",
				DataLimit:  DataSize(2000000000),
			},
		},
	}, store, &out)
	if err != nil {
		t.Fatalf("keys edit: %v", err)
	}

	// 6. Delete key by name
	out.Reset()
	err = execute(ctx, &Args{
		Keys: &KeysCmd{
			Delete: &DeleteKeyCmd{
				ServerName: "myserver",
				KeyName:    "Laptop",
			},
		},
	}, store, &out)
	if err != nil {
		t.Fatalf("keys delete: %v", err)
	}

	// 7. Metrics
	out.Reset()
	err = execute(ctx, &Args{
		Servers: &ServersCmd{
			Metrics: &MetricsCmd{ServerName: "myserver"},
		},
	}, store, &out)
	if err != nil {
		t.Fatalf("servers metrics: %v", err)
	}
	if !strings.Contains(out.String(), "User 1: 5.0 MB") {
		t.Errorf("unexpected metrics output:\n%s", out.String())
	}
}
