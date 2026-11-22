package config

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

type Server struct {
	Name       string `yaml:"name"`
	URL        string `yaml:"url"`
	CertSha256 string `yaml:"certSha256,omitempty"`
}

type fileData struct {
	Servers map[string]Server `yaml:"servers"`
}

// Store manages the local YAML configuration file of Outline servers.
type Store struct {
	path    string
	Servers map[string]Server
}

// DefaultPath returns ~/.config/outline-cli/config.yaml.
func DefaultPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "outline-cli", "config.yaml"), nil
}

// LoadDefault opens the configuration store at DefaultPath().
func LoadDefault() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return Load(path)
}

// Load reads the YAML configuration from path, creating the parent directory if needed.
// If the file does not exist yet, an empty Store is returned.
func Load(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	s := &Store{
		path:    path,
		Servers: make(map[string]Server),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var parsed fileData
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	if parsed.Servers != nil {
		s.Servers = parsed.Servers
	}

	return s, nil
}

func (s *Store) save() error {
	data, err := s.MarshalYAML()
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

func (s *Store) MarshalYAML() ([]byte, error) {
	data, err := yaml.Marshal(fileData{Servers: s.Servers})
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	return data, nil
}

func (s *Store) Get(name string) (Server, error) {
	srv, ok := s.Servers[name]
	if !ok {
		return Server{}, fmt.Errorf("server %q not found", name)
	}
	return srv, nil
}

func (s *Store) Add(name, serverURL, certSha256 string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("server name is required")
	}
	if _, exists := s.Servers[name]; exists {
		return fmt.Errorf("server %q already exists", name)
	}

	cleanURL, err := ValidateServerURL(serverURL)
	if err != nil {
		return err
	}
	cleanCert, err := ValidateCertSHA256(certSha256)
	if err != nil {
		return err
	}

	s.Servers[name] = Server{
		Name:       name,
		URL:        cleanURL,
		CertSha256: cleanCert,
	}

	return s.save()
}

type serverJSONInput struct {
	APIURL     string `json:"apiUrl"`
	CertSha256 string `json:"certSha256"`
}

func (s *Store) AddFromJSON(name, rawJSON string) error {
	var input serverJSONInput
	if err := json.Unmarshal([]byte(rawJSON), &input); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}
	if strings.TrimSpace(input.APIURL) == "" {
		return errors.New("apiUrl is required in JSON")
	}
	if strings.TrimSpace(input.CertSha256) == "" {
		return errors.New("certSha256 is required in JSON")
	}
	return s.Add(name, input.APIURL, input.CertSha256)
}

func (s *Store) UpdateURL(name, serverURL string) error {
	srv, err := s.Get(name)
	if err != nil {
		return err
	}

	cleanURL, err := ValidateServerURL(serverURL)
	if err != nil {
		return err
	}

	srv.URL = cleanURL
	s.Servers[name] = srv
	return s.save()
}

func (s *Store) Delete(name string) error {
	if _, err := s.Get(name); err != nil {
		return err
	}
	delete(s.Servers, name)
	return s.save()
}

// ValidateServerURL trims and validates an Outline management API URL.
func ValidateServerURL(raw string) (string, error) {
	urlStr := strings.TrimSpace(raw)
	if urlStr == "" {
		return "", errors.New("URL cannot be empty")
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}
	if parsed.Scheme == "" {
		return "", errors.New("URL must include a scheme (e.g., https://)")
	}
	if parsed.Host == "" {
		return "", errors.New("URL must include a host")
	}

	return urlStr, nil
}

// ValidateCertSHA256 trims and validates a hex-encoded certificate SHA-256 hash.
func ValidateCertSHA256(raw string) (string, error) {
	hash := strings.TrimSpace(raw)
	if hash == "" {
		return "", errors.New("certificate SHA256 cannot be empty")
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return "", fmt.Errorf("invalid SHA256 hash format: %w", err)
	}
	return hash, nil
}
