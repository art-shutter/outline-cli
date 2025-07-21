package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Servers map[string]Server `yaml:"servers"`
}

type Server struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type ConfigManager struct {
	configPath string
	config     *Config
	apiClient  *APIClient
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
		apiClient:  NewAPIClient(),
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
		fmt.Println("---")
	}

	return nil
}

func (cm *ConfigManager) AddServer(name, url string) error {
	if _, exists := cm.config.Servers[name]; exists {
		slog.Error("server already exists", "name", name)
		return fmt.Errorf("server '%s' already exists", name)
	}

	cm.config.Servers[name] = Server{
		Name: name,
		URL:  url,
	}

	if err := cm.saveConfig(); err != nil {
		slog.Error("failed to save config", "error", err)
		return err
	}

	slog.Info("server added successfully", "name", name)
	return nil
}

func (cm *ConfigManager) GetServer(name string) error {
	server, exists := cm.config.Servers[name]
	if !exists {
		slog.Error("server not found", "name", name)
		return fmt.Errorf("server '%s' not found", name)
	}

	fmt.Printf("Server: %s\n", name)
	fmt.Printf("URL:   %s\n", server.URL)

	// Get server information from API
	serverInfo, err := cm.apiClient.GetServerInfo(server.URL)
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

	accessKeys, err := cm.apiClient.ListAccessKeys(server.URL)
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
			fmt.Printf("Data Limit: %d bytes\n", key.DataLimit.Bytes)
		}
		fmt.Println("---")
	}

	return nil
}

// CreateAccessKey creates a new access key on a server
func (cm *ConfigManager) CreateAccessKey(serverName, keyName, method string, port int) error {
	server, exists := cm.config.Servers[serverName]
	if !exists {
		slog.Error("server not found", "name", serverName)
		return fmt.Errorf("server '%s' not found", serverName)
	}

	req := CreateAccessKeyRequest{
		Method: method,
	}
	if keyName != "" {
		req.Name = keyName
	}
	if port > 0 {
		req.Port = port
	}

	accessKey, err := cm.apiClient.CreateAccessKey(server.URL, req)
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
		fmt.Printf("Data Limit: %d bytes\n", accessKey.DataLimit.Bytes)
	}

	return nil
}

func (cm *ConfigManager) DeleteAccessKey(serverName, keyID string) error {
	server, exists := cm.config.Servers[serverName]
	if !exists {
		slog.Error("server not found", "serverName", serverName)
		return fmt.Errorf("server '%s' not found", serverName)
	}

	err := cm.apiClient.DeleteAccessKey(server.URL, keyID)
	if err != nil {
		slog.Error("failed to delete access key", "error", err)
		return err
	}

	slog.Debug("access key deleted successfully", "serverName", serverName, "keyID", keyID)
	return nil
}

func (cm *ConfigManager) GetMetrics(serverName string) error {
	server, exists := cm.config.Servers[serverName]
	if !exists {
		slog.Error("server not found", "serverName", serverName)
		return fmt.Errorf("server '%s' not found", serverName)
	}

	metrics, err := cm.apiClient.GetTransferMetrics(server.URL)
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
		fmt.Printf("User %s: %d bytes (%.2f MB)\n", userID, bytes, float64(bytes)/1024/1024)
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
