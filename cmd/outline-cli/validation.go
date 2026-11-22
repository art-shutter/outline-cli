package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/art-shutter/outline-cli/internal/config"
	"github.com/dustin/go-humanize"
)

func validateArgs(args *Args) error {
	if args.Servers != nil && args.Servers.Update != nil {
		if args.Servers.Update.URL == "" {
			return errors.New("--url must be specified for update operation")
		}
	}

	if args.Keys != nil {
		if args.Keys.Delete != nil {
			if args.Keys.Delete.KeyID == "" && args.Keys.Delete.KeyName == "" {
				return errors.New("either --key-id or --key-name must be specified for delete operation")
			}
		}

		if args.Keys.Edit != nil {
			if args.Keys.Edit.KeyID == "" && args.Keys.Edit.KeyName == "" {
				return errors.New("either --key-id or --key-name must be specified for edit operation")
			}
			if args.Keys.Edit.NewName == "" && args.Keys.Edit.DataLimit == 0 && !args.Keys.Edit.RemoveLimit {
				return errors.New("at least one of --new-name, --data-limit, or --remove-limit must be specified for edit operation")
			}
			if args.Keys.Edit.DataLimit > 0 && args.Keys.Edit.RemoveLimit {
				return errors.New("--data-limit and --remove-limit cannot be used together")
			}
		}
	}

	return nil
}

// DataSize represents a byte count parsed from a human-readable size flag (e.g. "1GB", "500MB").
type DataSize int64

func (d *DataSize) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*d = 0
		return nil
	}

	sizeStr := strings.TrimSpace(string(text))
	bytes, err := humanize.ParseBytes(sizeStr)
	if err != nil {
		return fmt.Errorf("invalid data size format (expected format like '1GB', '500MB', '2TB', got %q)", sizeStr)
	}

	*d = DataSize(bytes)
	return nil
}

func (d DataSize) MarshalText() ([]byte, error) {
	if d <= 0 {
		return []byte(""), nil
	}
	return []byte(humanize.Bytes(uint64(d))), nil
}

func (d DataSize) String() string {
	if d <= 0 {
		return ""
	}
	return humanize.Bytes(uint64(d))
}

// ServerURL represents a validated Outline management API URL.
type ServerURL string

func (s *ServerURL) UnmarshalText(text []byte) error {
	clean, err := config.ValidateServerURL(string(text))
	if err != nil {
		return err
	}
	*s = ServerURL(clean)
	return nil
}

func (s ServerURL) MarshalText() ([]byte, error) {
	return []byte(s), nil
}

func (s ServerURL) String() string {
	return string(s)
}

// CertSHA256 represents a validated hex-encoded certificate SHA-256 fingerprint.
type CertSHA256 string

func (c *CertSHA256) UnmarshalText(text []byte) error {
	clean, err := config.ValidateCertSHA256(string(text))
	if err != nil {
		return err
	}
	*c = CertSHA256(clean)
	return nil
}

func (c CertSHA256) MarshalText() ([]byte, error) {
	return []byte(c), nil
}

func (c CertSHA256) String() string {
	return string(c)
}

// Port represents a validated TCP/UDP port number in [1, 65535].
type Port int

func (p *Port) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return errors.New("port cannot be empty")
	}

	portStr := strings.TrimSpace(string(text))
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got: %d", port)
	}

	*p = Port(port)
	return nil
}

func (p Port) MarshalText() ([]byte, error) {
	return []byte(strconv.Itoa(int(p))), nil
}

func (p Port) String() string {
	return strconv.Itoa(int(p))
}

// EncryptionMethod represents a supported Shadowsocks AEAD cipher.
type EncryptionMethod string

const (
	MethodAES256GCM        EncryptionMethod = "aes-256-gcm"
	MethodAES192GCM        EncryptionMethod = "aes-192-gcm"
	MethodAES128GCM        EncryptionMethod = "aes-128-gcm"
	MethodChaCha20Poly1305 EncryptionMethod = "chacha20-poly1305"
)

func (e *EncryptionMethod) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*e = MethodAES192GCM
		return nil
	}

	method := EncryptionMethod(strings.TrimSpace(string(text)))
	switch method {
	case MethodAES256GCM, MethodAES192GCM, MethodAES128GCM, MethodChaCha20Poly1305:
		*e = method
		return nil
	default:
		return fmt.Errorf("invalid encryption method %q (valid: aes-256-gcm, aes-192-gcm, aes-128-gcm, chacha20-poly1305)", method)
	}
}

func (e EncryptionMethod) MarshalText() ([]byte, error) {
	return []byte(e), nil
}

func (e EncryptionMethod) String() string {
	return string(e)
}
