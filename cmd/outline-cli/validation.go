package main

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
)

func validateArgs(args *Args) error {
	if args.Keys != nil {
		if args.Keys.Delete != nil {
			if args.Keys.Delete.KeyID == "" && args.Keys.Delete.KeyName == "" {
				return fmt.Errorf("either --key-id or --key-name must be specified for delete operation")
			}
		}

		if args.Keys.Edit != nil {
			if args.Keys.Edit.KeyID == "" && args.Keys.Edit.KeyName == "" {
				return fmt.Errorf("either --key-id or --key-name must be specified for edit operation")
			}

			if args.Keys.Edit.NewName == "" && args.Keys.Edit.DataLimit.String() == "" && !args.Keys.Edit.RemoveLimit {
				return fmt.Errorf("at least one of --new-name, --data-limit, or --remove-limit must be specified for edit operation")
			}
		}
	}

	return nil
}

type DataSize struct {
	Bytes int64
}

func (d *DataSize) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		d.Bytes = 0
		return nil
	}

	sizeStr := strings.TrimSpace(string(text))

	bytes, err := humanize.ParseBytes(sizeStr)
	if err != nil {
		slog.Error("invalid data size format", "error", err, "size", sizeStr, "expected", "like 1GB, 500MB, 2TB", "got", sizeStr)
		return fmt.Errorf("invalid data size format. Expected format like '1GB', '500MB', '2TB'. Got: %s", sizeStr)
	}

	d.Bytes = int64(bytes)
	return nil
}

func (d DataSize) MarshalText() ([]byte, error) {
	if d.Bytes == 0 {
		return []byte(""), nil
	}
	return []byte(humanize.Bytes(uint64(d.Bytes))), nil
}

func (d DataSize) String() string {
	if d.Bytes == 0 {
		return ""
	}
	return humanize.Bytes(uint64(d.Bytes))
}

type ServerURL struct {
	URL string
}

func (s *ServerURL) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		slog.Error("URL cannot be empty")
		return fmt.Errorf("URL cannot be empty")
	}

	urlStr := strings.TrimSpace(string(text))

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		slog.Error("invalid URL format", "error", err, "url", urlStr)
		return fmt.Errorf("invalid URL format: %v", err)
	}

	if parsedURL.Scheme == "" {
		slog.Error("URL must include a scheme (e.g., https://)")
		return fmt.Errorf("URL must include a scheme (e.g., https://)")
	}

	if parsedURL.Host == "" {
		slog.Error("URL must include a host")
		return fmt.Errorf("URL must include a host")
	}

	s.URL = urlStr
	return nil
}

func (s ServerURL) MarshalText() ([]byte, error) {
	return []byte(s.URL), nil
}

func (s ServerURL) String() string {
	return s.URL
}

type CertSHA256 struct {
	Hash string
}

func (c *CertSHA256) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return fmt.Errorf("certificate SHA256 cannot be empty")
	}

	hash := strings.TrimSpace(string(text))

	_, err := hex.DecodeString(hash)
	if err != nil {
		slog.Error("invalid SHA256 hash format", "error", err, "hash", hash)
		return fmt.Errorf("invalid SHA256 hash format: %v", err)
	}

	c.Hash = hash
	return nil
}

func (c CertSHA256) MarshalText() ([]byte, error) {
	return []byte(c.Hash), nil
}

func (c CertSHA256) String() string {
	return c.Hash
}

type Port struct {
	Number int
}

func (p *Port) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		slog.Error("port cannot be empty")
		return fmt.Errorf("port cannot be empty")
	}

	portStr := strings.TrimSpace(string(text))

	port, err := strconv.Atoi(portStr)
	if err != nil {
		slog.Error("invalid port number", "error", err, "port", portStr)
		return fmt.Errorf("invalid port number: %v", err)
	}

	if port < 1 || port > 65535 {
		slog.Error("port must be between 1 and 65535", "port", port)
		return fmt.Errorf("port must be between 1 and 65535, got: %d", port)
	}

	p.Number = port
	return nil
}

func (p Port) MarshalText() ([]byte, error) {
	return []byte(strconv.Itoa(p.Number)), nil
}

func (p Port) String() string {
	return strconv.Itoa(p.Number)
}

type EncryptionMethod struct {
	Method string
}

var validEncryptionMethods = map[string]bool{
	"aes-256-gcm":       true,
	"aes-192-gcm":       true,
	"aes-128-gcm":       true,
	"chacha20-poly1305": true,
}

func (e *EncryptionMethod) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		e.Method = "aes-192-gcm"
		return nil
	}

	method := strings.TrimSpace(string(text))

	if !validEncryptionMethods[method] {
		validMethods := make([]string, 0, len(validEncryptionMethods))
		for m := range validEncryptionMethods {
			validMethods = append(validMethods, m)
		}
		slog.Error("invalid encryption method", "method", method, "valid_methods", strings.Join(validMethods, ", "))
		return fmt.Errorf("invalid encryption method. Valid methods are: %s", strings.Join(validMethods, ", "))
	}

	e.Method = method
	return nil
}

func (e EncryptionMethod) MarshalText() ([]byte, error) {
	return []byte(e.Method), nil
}

func (e EncryptionMethod) String() string {
	return e.Method
}

func ParseDataSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, nil
	}

	sizeStr = strings.TrimSpace(sizeStr)

	bytes, err := humanize.ParseBytes(sizeStr)
	if err != nil {
		slog.Error("invalid data size format", "error", err, "size", sizeStr, "expected", "like 1GB, 500MB, 2TB", "got", sizeStr)
		return 0, fmt.Errorf("invalid data size format. Expected format like '1GB', '500MB', '2TB'. Got: %s", sizeStr)
	}

	return int64(bytes), nil
}
