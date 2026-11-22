package config

import (
	"path/filepath"
	"testing"
)

func TestStoreLifecycle(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "outline-cli", "config.yaml")

	store, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load empty store: %v", err)
	}
	if len(store.Servers) != 0 {
		t.Fatalf("expected 0 servers, got %d", len(store.Servers))
	}

	cert := "34B3C8EB1C6EC9B5335556D7E8DC73A30152D27C66B054BAB8ACF5D11AE0C810"
	if err := store.Add("prod", "https://example.com/secret", cert); err != nil {
		t.Fatalf("Add server: %v", err)
	}

	// Duplicate add should fail
	if err := store.Add("prod", "https://example.com/secret", cert); err == nil {
		t.Fatal("expected error when adding duplicate server")
	}

	// Reload from disk to verify persistence
	reloaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Reload store: %v", err)
	}
	srv, err := reloaded.Get("prod")
	if err != nil {
		t.Fatalf("Get prod: %v", err)
	}
	if srv.URL != "https://example.com/secret" || srv.CertSha256 != cert {
		t.Errorf("unexpected server persisted: %+v", srv)
	}

	// Update URL
	if err := reloaded.UpdateURL("prod", "https://new.example.com/secret"); err != nil {
		t.Fatalf("UpdateURL: %v", err)
	}
	srv, err = reloaded.Get("prod")
	if err != nil {
		t.Fatalf("Get prod after update: %v", err)
	}
	if srv.URL != "https://new.example.com/secret" {
		t.Errorf("URL = %q, want %q", srv.URL, "https://new.example.com/secret")
	}

	// Add from JSON
	jsonPayload := `{"apiUrl":"https://vpn.example.com:60000/path","certSha256":"` + cert + `"}`
	if err := reloaded.AddFromJSON("from-json", jsonPayload); err != nil {
		t.Fatalf("AddFromJSON: %v", err)
	}
	if _, err := reloaded.Get("from-json"); err != nil {
		t.Fatalf("Get from-json: %v", err)
	}

	// Delete server
	if err := reloaded.Delete("prod"); err != nil {
		t.Fatalf("Delete prod: %v", err)
	}
	if _, err := reloaded.Get("prod"); err == nil {
		t.Fatal("expected error getting deleted server")
	}
}

func TestAddFromJSON_Validation(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	tests := []struct {
		name string
		json string
	}{
		{"invalid json", `{not-json}`},
		{"missing apiUrl", `{"certSha256":"1234"}`},
		{"missing certSha256", `{"apiUrl":"https://example.com/path"}`},
		{"invalid apiUrl", `{"apiUrl":"not-a-url","certSha256":"1234"}`},
		{"invalid certSha256", `{"apiUrl":"https://example.com/path","certSha256":"not-hex"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.AddFromJSON("srv", tt.json); err == nil {
				t.Errorf("AddFromJSON(%s) expected error, got nil", tt.json)
			}
		})
	}
}
