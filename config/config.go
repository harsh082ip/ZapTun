// In pkg/config/config.go

package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	ServerConfigFilePath = "server_config.json"
	ClientConfigFilePath = "client_config.json"
)

// ServerConfig holds all configuration for the tunnel-server.
type ServerConfig struct {
	Domain           string `json:"domain"`             // e.g., "zaptun.com"
	ControlPlaneAddr string `json:"control_plane_addr"` // e.g., ":4443"
	DataPlaneAddr    string `json:"data_plane_addr"`    // e.g., ":80"
	LogFile          string `json:"log_file"`           // e.g., "/var/log/zaptun/server.log"
	LogLevel         string `json:"log_level"`          // e.g., "info", "debug"
}

// ClientConfig holds all configuration for the tunnel-client.
type ClientConfig struct {
	ServerAddr string `json:"server_addr"` // e.g., "zaptun.com:4443"
	AuthToken  string `json:"auth_token"`  // The token to authenticate with the server
}

// LoadServerConfig loads server configuration from a given JSON file path.
func LoadServerConfig(path string) (*ServerConfig, error) {
	if path == "" {
		path = ServerConfigFilePath
	}
	if !fileExists(path) {
		// Return an error if the file doesn't exist
		return &ServerConfig{}, fmt.Errorf("server config needed to start the server")
	}

	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ServerConfig
	err = json.Unmarshal(f, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadClientConfig loads client configuration from a given JSON file path.
func LoadClientConfig(path string) (*ClientConfig, error) {
	if path == "" {
		path = ClientConfigFilePath
	}
	if !fileExists(path) {
		// Return a default configuration if the file doesn't exist
		return &ClientConfig{
			ServerAddr: "example.com:4443",
			AuthToken:  "",
		}, nil
	}

	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ClientConfig
	err = json.Unmarshal(f, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
