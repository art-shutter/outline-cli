package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/goccy/go-yaml"

	"github.com/art-shutter/outline-cli/internal/api"
)

type Config struct {
	Servers map[string]Server `yaml:"servers"`
}

type Server struct {
	Name       string `yaml:"name"`
	URL        string `yaml:"url"`
	CertSha256 string `yaml:"certSha256,omitempty"`
}

type ConfigManager struct {
	configPath string
	config     *Config
}

func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("failed to get home directory", "error", err)
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".config", "outline-cli")
	configPath := filepath.Join(configDir, "config.yaml")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		slog.Error("failed to create config directory", "error", err)
		return nil, err
	}

	cm := &ConfigManager{
		configPath: configPath,
		config:     &Config{Servers: make(map[string]Server)},
	}

	if err := cm.loadConfig(); err != nil {
		slog.Error("failed to load config", "error", err)
		return nil, err
	}

	return cm, nil
}

func (cm *ConfigManager) loadConfig() error {
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		slog.Warn("config file does not exist", "path", cm.configPath)
		return nil
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		slog.Error("failed to read config file", "error", err)
		return err
	}

	if err := yaml.Unmarshal(data, cm.config); err != nil {
		slog.Error("failed to parse config file", "error", err)
		return err
	}

	if cm.config.Servers == nil {
		slog.Debug("config file is empty, creating default config")
		cm.config.Servers = make(map[string]Server)
	}

	return nil
}

func (cm *ConfigManager) saveConfig() error {
	data, err := yaml.Marshal(cm.config)
	if err != nil {
		slog.Error("failed to marshal config", "error", err)
		return err
	}

	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		slog.Error("failed to write config file", "error", err)
		return err
	}

	return nil
}

func (cm *ConfigManager) ListServers() error {
	if len(cm.config.Servers) == 0 {
		slog.Debug("no servers configured")
		return nil
	}

	fmt.Println("Configured servers:")
	fmt.Println("===================")
	for name, server := range cm.config.Servers {
		fmt.Printf("Name: %s\n", name)
		fmt.Printf("URL:  %s\n", server.URL)
		fmt.Printf("Cert: %s\n", server.CertSha256)
		fmt.Println("---")
	}

	return nil
}

func (cm *ConfigManager) AddServer(name, url, certSha256 string) error {
	if _, exists := cm.config.Servers[name]; exists {
		slog.Error("server already exists", "name", name)
		return fmt.Errorf("server '%s' already exists", name)
	}

	if certSha256 == "" {
		return fmt.Errorf("certificate SHA256 is required")
	}

	cm.config.Servers[name] = Server{
		Name:       name,
		URL:        url,
		CertSha256: certSha256,
	}

	if err := cm.saveConfig(); err != nil {
		slog.Error("failed to save config", "error", err)
		return err
	}

	slog.Info("server added successfully", "name", name)
	return nil
}

// getAPIClientForServer returns an API client configured for the specified server
func (cm *ConfigManager) getAPIClientForServer(serverName string) (*api.APIClient, error) {
	server, exists := cm.config.Servers[serverName]
	if !exists {
		return nil, fmt.Errorf("server '%s' not found", serverName)
	}

	return api.NewAPIClient(server.CertSha256), nil
}

// AddServerFromJSON adds a server from JSON input
func (cm *ConfigManager) AddServerFromJSON(serverName, jsonInput string) error {
	var serverData struct {
		APIURL     string `json:"apiUrl"`
		CertSha256 string `json:"certSha256"`
	}

	if err := json.Unmarshal([]byte(jsonInput), &serverData); err != nil {
		slog.Error("failed to parse JSON input", "error", err)
		return fmt.Errorf("invalid JSON format: %v", err)
	}

	if serverData.APIURL == "" {
		return fmt.Errorf("apiUrl is required in JSON")
	}
	if serverData.CertSha256 == "" {
		return fmt.Errorf("certSha256 is required in JSON")
	}

	return cm.AddServer(serverName, serverData.APIURL, serverData.CertSha256)
}

func (cm *ConfigManager) GetServer(name string) error {
	server, exists := cm.config.Servers[name]
	if !exists {
		slog.Error("server not found", "name", name)
		return fmt.Errorf("server '%s' not found", name)
	}

	fmt.Printf("Server: %s\n", name)
	fmt.Printf("URL:   %s\n", server.URL)
	if server.CertSha256 != "" {
		fmt.Printf("Cert:  %s\n", server.CertSha256)
	}

	// Get API client for this server
	apiClient, err := cm.getAPIClientForServer(name)
	if err != nil {
		slog.Error("failed to get API client", "error", err)
		return err
	}

	// Get server information from API
	serverInfo, err := apiClient.GetServerInfo(server.URL)
	if err != nil {
		slog.Warn("failed to get server info from API", "error", err)
		return nil
	}

	fmt.Printf("API Info:\n")
	fmt.Printf("  Name:                    %s\n", serverInfo.Name)
	fmt.Printf("  Server ID:               %s\n", serverInfo.ServerID)
	fmt.Printf("  Version:                 %s\n", serverInfo.Version)
	fmt.Printf("  Metrics Enabled:         %t\n", serverInfo.MetricsEnabled)
	fmt.Printf("  Port for New Keys:       %d\n", serverInfo.PortForNewAccessKeys)
	fmt.Printf("  Hostname for Keys:       %s\n", serverInfo.HostnameForAccessKeys)
	if serverInfo.AccessKeyDataLimit != nil {
		fmt.Printf("  Access Key Data Limit:   %d bytes\n", serverInfo.AccessKeyDataLimit.Bytes)
	}
	return nil
}

func (cm *ConfigManager) UpdateServer(name, url string) error {
	server, exists := cm.config.Servers[name]
	if !exists {
		slog.Error("server not found", "name", name)
		return fmt.Errorf("server '%s' not found", name)
	}

	if url != "" {
		slog.Debug("updating server URL", "name", name, "url", url)
		server.URL = url
		cm.config.Servers[name] = server
	}

	if err := cm.saveConfig(); err != nil {
		slog.Error("failed to save config", "error", err)
		return err
	}

	slog.Debug("server updated successfully", "name", name)
	return nil
}

func (cm *ConfigManager) DeleteServer(name string) error {
	if _, exists := cm.config.Servers[name]; !exists {
		slog.Error("server not found", "name", name)
		return fmt.Errorf("server '%s' not found", name)
	}

	delete(cm.config.Servers, name)

	if err := cm.saveConfig(); err != nil {
		slog.Error("failed to save config", "error", err)
		return err
	}

	slog.Debug("server deleted successfully", "name", name)
	return nil
}

func (cm *ConfigManager) ListAccessKeys(serverName string) error {
	server, exists := cm.config.Servers[serverName]
	if !exists {
		slog.Error("server not found", "name", serverName)
		return fmt.Errorf("server '%s' not found", serverName)
	}

	// Get API client for this server
	apiClient, err := cm.getAPIClientForServer(serverName)
	if err != nil {
		slog.Error("failed to get API client", "error", err)
		return err
	}

	accessKeys, err := apiClient.ListAccessKeys(server.URL)
	if err != nil {
		slog.Error("failed to list access keys", "error", err)
		return err
	}

	if len(accessKeys) == 0 {
		slog.Debug("no access keys found on server", "name", serverName)
		return nil
	}

	fmt.Printf("Access keys for server '%s':\n", serverName)
	fmt.Println("==================================")
	for _, key := range accessKeys {
		fmt.Printf("ID:       %s\n", key.ID)
		fmt.Printf("Name:     %s\n", key.Name)
		fmt.Printf("Port:     %d\n", key.Port)
		fmt.Printf("Method:   %s\n", key.Method)
		fmt.Printf("Access URL: %s\n", key.AccessURL)
		if key.DataLimit != nil {
			fmt.Printf("Data Limit: %s\n", humanize.Bytes(uint64(key.DataLimit.Bytes)))
		}
		fmt.Println("---")
	}

	return nil
}

// CreateAccessKey creates a new access key on a server
func (cm *ConfigManager) CreateAccessKey(serverName, keyName, method string, port int, dataLimitStr string) error {
	server, exists := cm.config.Servers[serverName]
	if !exists {
		slog.Error("server not found", "name", serverName)
		return fmt.Errorf("server '%s' not found", serverName)
	}

	// Parse data limit if provided
	var dataLimit int64
	if dataLimitStr != "" {
		var err error
		dataLimit, err = ParseDataSize(dataLimitStr)
		if err != nil {
			slog.Error("failed to parse data limit", "error", err)
			return err
		}
	}

	req := api.CreateAccessKeyRequest{
		Method: method,
	}
	if keyName != "" {
		req.Name = keyName
	}
	if port > 0 {
		req.Port = port
	}
	if dataLimit > 0 {
		req.Limit = &api.DataLimit{Bytes: dataLimit}
	}

	// Get API client for this server
	apiClient, err := cm.getAPIClientForServer(serverName)
	if err != nil {
		slog.Error("failed to get API client", "error", err)
		return err
	}

	accessKey, err := apiClient.CreateAccessKey(server.URL, req)
	if err != nil {
		slog.Error("failed to create access key", "error", err)
		return err
	}

	fmt.Printf("Access key created successfully!\n")
	fmt.Printf("ID:         %s\n", accessKey.ID)
	fmt.Printf("Name:       %s\n", accessKey.Name)
	fmt.Printf("Password:   %s\n", accessKey.Password)
	fmt.Printf("Port:       %d\n", accessKey.Port)
	fmt.Printf("Method:     %s\n", accessKey.Method)
	fmt.Printf("Access URL: %s\n", accessKey.AccessURL)
	if accessKey.DataLimit != nil {
		fmt.Printf("Data Limit: %s\n", humanize.Bytes(uint64(accessKey.DataLimit.Bytes)))
	}

	return nil
}

func (cm *ConfigManager) DeleteAccessKey(serverName, keyID string) error {
	server, exists := cm.config.Servers[serverName]
	if !exists {
		slog.Error("server not found", "serverName", serverName)
		return fmt.Errorf("server '%s' not found", serverName)
	}

	// Get API client for this server
	apiClient, err := cm.getAPIClientForServer(serverName)
	if err != nil {
		slog.Error("failed to get API client", "error", err)
		return err
	}

	err = apiClient.DeleteAccessKey(server.URL, keyID)
	if err != nil {
		slog.Error("failed to delete access key", "error", err)
		return err
	}

	slog.Debug("access key deleted successfully", "serverName", serverName, "keyID", keyID)
	return nil
}

// DeleteAccessKeyByName deletes an access key by name
func (cm *ConfigManager) DeleteAccessKeyByName(serverName, keyName string) error {
	server, exists := cm.config.Servers[serverName]
	if !exists {
		slog.Error("server not found", "serverName", serverName)
		return fmt.Errorf("server '%s' not found", serverName)
	}

	// Get API client for this server
	apiClient, err := cm.getAPIClientForServer(serverName)
	if err != nil {
		slog.Error("failed to get API client", "error", err)
		return err
	}

	// First, get all access keys to find the one with the matching name
	accessKeys, err := apiClient.ListAccessKeys(server.URL)
	if err != nil {
		slog.Error("failed to list access keys", "error", err)
		return err
	}

	var keyID string
	for _, key := range accessKeys {
		if key.Name == keyName {
			keyID = key.ID
			break
		}
	}

	if keyID == "" {
		slog.Error("access key not found", "serverName", serverName, "keyName", keyName)
		return fmt.Errorf("access key with name '%s' not found on server '%s'", keyName, serverName)
	}

	return cm.DeleteAccessKey(serverName, keyID)
}

func (cm *ConfigManager) GetMetrics(serverName string) error {
	server, exists := cm.config.Servers[serverName]
	if !exists {
		slog.Error("server not found", "serverName", serverName)
		return fmt.Errorf("server '%s' not found", serverName)
	}

	// Get API client for this server
	apiClient, err := cm.getAPIClientForServer(serverName)
	if err != nil {
		slog.Error("failed to get API client", "error", err)
		return err
	}

	metrics, err := apiClient.GetTransferMetrics(server.URL)
	if err != nil {
		slog.Error("failed to get metrics", "error", err)
		return err
	}

	fmt.Printf("Transfer metrics for server '%s':\n", serverName)
	fmt.Println("==================================")
	if len(metrics.BytesTransferredByUserId) == 0 {
		slog.Debug("no transfer data available", "serverName", serverName)
		return nil
	}

	for userID, bytes := range metrics.BytesTransferredByUserId {
		fmt.Printf("User %s: %s\n", userID, humanize.Bytes(uint64(bytes)))
	}

	return nil
}

func (cm *ConfigManager) PrintConfig() error {
	data, err := yaml.Marshal(cm.config)
	if err != nil {
		slog.Error("failed to marshal config", "error", err)
		return err
	}

	fmt.Println(string(data))
	return nil
}

// EditAccessKey edits an existing access key
func (cm *ConfigManager) EditAccessKey(serverName, keyID, keyName, newName, dataLimitStr string, removeLimit bool) error {
	server, exists := cm.config.Servers[serverName]
	if !exists {
		slog.Error("server not found", "serverName", serverName)
		return fmt.Errorf("server '%s' not found", serverName)
	}

	// Get API client for this server
	apiClient, err := cm.getAPIClientForServer(serverName)
	if err != nil {
		slog.Error("failed to get API client", "error", err)
		return err
	}

	// Determine the actual key ID
	actualKeyID := keyID
	if keyName != "" {
		// Find key by name
		accessKeys, err := apiClient.ListAccessKeys(server.URL)
		if err != nil {
			slog.Error("failed to list access keys", "error", err)
			return err
		}

		found := false
		for _, key := range accessKeys {
			if key.Name == keyName {
				actualKeyID = key.ID
				found = true
				break
			}
		}

		if !found {
			slog.Error("access key not found", "serverName", serverName, "keyName", keyName)
			return fmt.Errorf("access key with name '%s' not found on server '%s'", keyName, serverName)
		}
	}

	if actualKeyID == "" {
		return fmt.Errorf("either --key-id or --key-name must be specified")
	}

	// Update key name if provided
	if newName != "" {
		err := apiClient.RenameAccessKey(server.URL, actualKeyID, newName)
		if err != nil {
			slog.Error("failed to rename access key", "error", err)
			return err
		}
		fmt.Printf("Access key renamed successfully to: %s\n", newName)
	}

	// Handle data limit changes
	if removeLimit {
		err := apiClient.RemoveAccessKeyDataLimit(server.URL, actualKeyID)
		if err != nil {
			slog.Error("failed to remove data limit", "error", err)
			return err
		}
		fmt.Printf("Data limit removed successfully\n")
	} else if dataLimitStr != "" {
		// Parse and set new data limit
		dataLimit, err := ParseDataSize(dataLimitStr)
		if err != nil {
			slog.Error("failed to parse data limit", "error", err)
			return err
		}

		err = apiClient.SetAccessKeyDataLimit(server.URL, actualKeyID, api.DataLimit{Bytes: dataLimit})
		if err != nil {
			slog.Error("failed to set data limit", "error", err)
			return err
		}
		fmt.Printf("Data limit updated successfully to: %s\n", humanize.Bytes(uint64(dataLimit)))
	}

	return nil
}

// ParseDataSize parses human-readable data sizes using go-humanize library
func ParseDataSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, nil
	}

	// Remove any whitespace
	sizeStr = strings.TrimSpace(sizeStr)

	bytes, err := humanize.ParseBytes(sizeStr)
	if err != nil {
		return 0, fmt.Errorf("invalid data size format. Expected format like '1GB', '500MB', '2TB'. Got: %s", sizeStr)
	}

	return int64(bytes), nil
}
