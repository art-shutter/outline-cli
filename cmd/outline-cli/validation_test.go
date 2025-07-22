package main

import (
	"testing"
)

func TestDataSize_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
		hasError bool
	}{
		{"empty string", "", 0, false},
		{"1 byte", "1B", 1, false},
		{"1 kilobyte", "1KB", 1000, false},
		{"1 megabyte", "1MB", 1000000, false},
		{"1 gigabyte", "1GB", 1000000000, false},
		{"1 terabyte", "1TB", 1000000000000, false},
		{"decimal values", "1.5GB", 1500000000, false},
		{"with spaces", " 1GB ", 1000000000, false},
		{"lowercase", "1gb", 1000000000, false},
		{"mixed case", "1Gb", 1000000000, false},
		{"kibibyte", "1KiB", 1024, false},
		{"mebibyte", "1MiB", 1048576, false},
		{"gibibyte", "1GiB", 1073741824, false},
		{"tebibyte", "1TiB", 1099511627776, false},
		{"zero", "0GB", 0, false},
		{"number without unit", "1000", 1000, false},

		// Invalid inputs
		{"invalid format", "invalid", 0, true},
		{"unknown unit", "1ZB", 0, true},
		{"negative number", "-1GB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ds DataSize
			err := ds.UnmarshalText([]byte(tt.input))

			if tt.hasError {
				if err == nil {
					t.Errorf("DataSize.UnmarshalText(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("DataSize.UnmarshalText(%q) unexpected error: %v", tt.input, err)
				}
				if ds.Bytes != tt.expected {
					t.Errorf("DataSize.UnmarshalText(%q) = %d, want %d", tt.input, ds.Bytes, tt.expected)
				}
			}
		})
	}
}

func TestServerURL_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"valid https URL", "https://example.com/secret", "https://example.com/secret", false},
		{"valid http URL", "http://localhost:8080/secret", "http://localhost:8080/secret", false},
		{"URL with query params", "https://example.com/secret?param=value", "https://example.com/secret?param=value", false},
		{"URL with port", "https://example.com:8443/secret", "https://example.com:8443/secret", false},
		{"URL with spaces", " https://example.com/secret ", "https://example.com/secret", false},

		// Invalid inputs
		{"empty string", "", "", true},
		{"missing scheme", "example.com/secret", "", true},
		{"missing host", "https:///secret", "", true},
		{"invalid URL", "not-a-url", "", true},
		{"only scheme", "https://", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var su ServerURL
			err := su.UnmarshalText([]byte(tt.input))

			if tt.hasError {
				if err == nil {
					t.Errorf("ServerURL.UnmarshalText(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ServerURL.UnmarshalText(%q) unexpected error: %v", tt.input, err)
				}
				if su.URL != tt.expected {
					t.Errorf("ServerURL.UnmarshalText(%q) = %q, want %q", tt.input, su.URL, tt.expected)
				}
			}
		})
	}
}

func TestCertSHA256_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"valid hex string", "1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF", "1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF", false},
		{"valid hex string lowercase", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
		{"valid hex string mixed case", "1234567890ABCDEF1234567890abcdef1234567890ABCDEF1234567890abcdef", "1234567890ABCDEF1234567890abcdef1234567890ABCDEF1234567890abcdef", false},
		{"with spaces", " 1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF ", "1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF", false},

		// Invalid inputs
		{"empty string", "", "", true},
		{"invalid characters", "1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEG", "", true},
		{"not hex", "not-a-hex-string-not-a-hex-string-not-a-hex-string-not-a-hex", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cs CertSHA256
			err := cs.UnmarshalText([]byte(tt.input))

			if tt.hasError {
				if err == nil {
					t.Errorf("CertSHA256.UnmarshalText(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("CertSHA256.UnmarshalText(%q) unexpected error: %v", tt.input, err)
				}
				if cs.Hash != tt.expected {
					t.Errorf("CertSHA256.UnmarshalText(%q) = %q, want %q", tt.input, cs.Hash, tt.expected)
				}
			}
		})
	}
}

func TestPort_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{"valid port 1", "1", 1, false},
		{"valid port 1024", "1024", 1024, false},
		{"valid port 65535", "65535", 65535, false},
		{"with spaces", " 8080 ", 8080, false},

		// Invalid inputs
		{"empty string", "", 0, true},
		{"zero", "0", 0, true},
		{"negative", "-1", 0, true},
		{"too large", "65536", 0, true},
		{"not a number", "not-a-number", 0, true},
		{"decimal", "8080.5", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p Port
			err := p.UnmarshalText([]byte(tt.input))

			if tt.hasError {
				if err == nil {
					t.Errorf("Port.UnmarshalText(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Port.UnmarshalText(%q) unexpected error: %v", tt.input, err)
				}
				if p.Number != tt.expected {
					t.Errorf("Port.UnmarshalText(%q) = %d, want %d", tt.input, p.Number, tt.expected)
				}
			}
		})
	}
}

func TestValidateArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    *Args
		wantErr bool
	}{
		{
			name:    "valid args - no keys command",
			args:    &Args{},
			wantErr: false,
		},
		{
			name: "valid args - keys command with list",
			args: &Args{
				Keys: &KeysCmd{
					List: &ListKeysCmd{ServerName: ServerName{ServerName: "test"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid args - delete with key ID",
			args: &Args{
				Keys: &KeysCmd{
					Delete: &DeleteKeyCmd{
						ServerName: ServerName{ServerName: "test"},
						KeyID:      "key123",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid args - delete with key name",
			args: &Args{
				Keys: &KeysCmd{
					Delete: &DeleteKeyCmd{
						ServerName: ServerName{ServerName: "test"},
						KeyName:    "test-key",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid args - delete without key ID or name",
			args: &Args{
				Keys: &KeysCmd{
					Delete: &DeleteKeyCmd{
						ServerName: ServerName{ServerName: "test"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid args - edit with new name",
			args: &Args{
				Keys: &KeysCmd{
					Edit: &EditKeyCmd{
						ServerName: ServerName{ServerName: "test"},
						KeyID:      "key123",
						NewName:    "new-name",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid args - edit with data limit",
			args: &Args{
				Keys: &KeysCmd{
					Edit: &EditKeyCmd{
						ServerName: ServerName{ServerName: "test"},
						KeyID:      "key123",
						DataLimit:  DataSize{Bytes: 1024 * 1024 * 1024}, // 1GB
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid args - edit with remove limit",
			args: &Args{
				Keys: &KeysCmd{
					Edit: &EditKeyCmd{
						ServerName:  ServerName{ServerName: "test"},
						KeyID:       "key123",
						RemoveLimit: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid args - edit without key ID or name",
			args: &Args{
				Keys: &KeysCmd{
					Edit: &EditKeyCmd{
						ServerName: ServerName{ServerName: "test"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid args - edit without any changes",
			args: &Args{
				Keys: &KeysCmd{
					Edit: &EditKeyCmd{
						ServerName: ServerName{ServerName: "test"},
						KeyID:      "key123",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
