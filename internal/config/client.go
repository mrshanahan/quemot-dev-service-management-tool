package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type ClientConfig struct {
	DefaultServer string                              `json:"default_server"`
	Servers       map[string]*ClientServerConfigEntry `json:"servers"`
}

type ClientServerConfigEntry struct {
	Hostname             string `json:"hostname"`
	SshKeyFilePath       string `json:"ssh_key_file_path"`
	SshKeyFilePassphrase string `json:"ssh_key_file_passphrase"`
	SshUsername          string `json:"ssh_username"`
}

func GetDefaultClientConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user's home directory: %w", err)
	}
	configDirPath := filepath.Join(home, ".config")
	if err = os.MkdirAll(configDirPath, os.FileMode(os.O_RDWR)); err != nil {
		return "", fmt.Errorf("failed to validate default client config directory: %w", err)
	}

	configPath := filepath.Join(configDirPath, "smt.config")
	return configPath, nil
}

func LoadClientConfig(path string, force bool) (*ClientConfig, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !force {
				return nil, fmt.Errorf("client config file does not exist at %s: %w", path, err)
			}
			slog.Debug("client config file does not exist; creating", "path", path)
			defaultConfig := &ClientConfig{"", map[string]*ClientServerConfigEntry{}}
			if err := SaveClientConfig(path, defaultConfig); err != nil {
				return nil, fmt.Errorf("failed to initialize client config file: %w", err)
			}
			return defaultConfig, nil
		}
		return nil, fmt.Errorf("failed to read client config file: %w", err)
	}

	var config *ClientConfig
	if err = json.Unmarshal(contents, &config); err != nil {
		return nil, fmt.Errorf("failed to parse client config file: %w", err)
	}

	return config, nil
}

func SaveClientConfig(path string, c *ClientConfig) error {
	configJson, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize client config file: %w", err)
	}
	if err = os.WriteFile(path, configJson, 0744); err != nil {
		return fmt.Errorf("failed to write client config file: %w", err)
	}
	return nil
}
