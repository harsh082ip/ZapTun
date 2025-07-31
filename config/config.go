package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

const (
	ServerConfigFilePath = "server_config.json"
	ClientConfigFilePath = "client_config.json"
)

var localConfig = ".zaptun-config"
var remoteConfig = "https://zaptun.com/config.json"

type ServerConfig struct {
	Domain             string `json:"domain"`
	ControlPlaneAddr   string `json:"control_plane_addr"`
	DataPlaneAddr      string `json:"data_plane_addr"`
	LogFile            string `json:"log_file"`
	LogLevel           string `json:"log_level"`
	CertificatePath    string `json:"certificate_path"`
	PrivateKeyPath     string `json:"private_key_path"`
	GitHubClientID     string `json:"github_client_id"`
	GitHubClientSecret string `json:"github_client_secret"`
}

type ClientConfig struct {
	Remote struct {
		ServerAddr string `json:"server_addr"`
	}
	Local struct {
		AuthToken string `json:"auth_token"`
	}
}

func LoadServerConfig(path string) (*ServerConfig, error) {
	if path == "" {
		path = ServerConfigFilePath
	}
	if !fileExists(path) {
		return nil, fmt.Errorf("server config needed to start the server")
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

func LoadClientConfig() (*ClientConfig, error) {
	var c ClientConfig
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("error getting user config directory: %s", err)
	}
	filePath := filepath.Join(configDir, "zaptun", localConfig)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error: no auth token, obtain at https://zaptun.com/auth")
	}
	if err := json.Unmarshal(data, &c.Local); err != nil {
		return nil, fmt.Errorf("error unmarshaling config file contents: %s", err)
	}
	response, err := http.Get(remoteConfig)
	if err != nil || response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching %s: %s", remoteConfig, err)
	}
	defer response.Body.Close()

	if err := json.NewDecoder(response.Body).Decode(&c.Remote); err != nil {
		return nil, fmt.Errorf("error decoding config file: %s", err)
	}
	return &c, nil
}

func WriteAuthToken(token string) error {
	var c ClientConfig
	c.Local.AuthToken = token
	content, err := json.Marshal(c.Local)
	if err != nil {
		return fmt.Errorf("error marshaling config: %s", err)
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("error getting user config directory: %s", err)
	}
	dirPath := filepath.Join(configDir, "zaptun")
	if err := os.MkdirAll(dirPath, 0700); err != nil && os.IsNotExist(err) {
		return fmt.Errorf("error creating config directory: %s", err)
	}
	filePath := filepath.Join(dirPath, localConfig)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error creating config file: %s", err)
	}
	if _, err = file.Write(content); err != nil {
		return fmt.Errorf("error writitng to config file: %s", err)
	}
	return nil
}
