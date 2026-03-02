package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	DefaultRemoteServiceDirectory string = "/usr/local/smt"
)

type Config struct {
	DefaultServer string                   `json:"default_server"`
	Servers       map[string]*ServerConfig `json:"servers"`
}

type ServerConfig struct {
	Hostname               string `json:"hostname"`
	SshKeyFilePath         string `json:"ssh_key_file_path"`
	SshKeyFilePassphrase   string `json:"ssh_key_file_passphrase"`
	SshUsername            string `json:"ssh_username"`
	RemoteServiceDirectory string `json:"service_directory"`
}

func GetDefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user's home directory: %w", err)
	}
	configDirPath := filepath.Join(home, ".config")
	if err = os.MkdirAll(configDirPath, os.FileMode(os.O_RDWR)); err != nil {
		return "", fmt.Errorf("failed to validate default config directory: %w", err)
	}

	configPath := filepath.Join(configDirPath, "smt.config")
	return configPath, nil
}

func LoadConfig(path string, force bool) (*Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !force {
				return nil, fmt.Errorf("config file does not exist at %s: %w", path, err)
			}
			slog.Debug("config file does not exist; creating", "path", path)
			defaultConfig := &Config{"", map[string]*ServerConfig{}}
			if err := SaveConfig(path, defaultConfig); err != nil {
				return nil, fmt.Errorf("failed to initialize config file: %w", err)
			}
			return defaultConfig, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config *Config
	if err = json.Unmarshal(contents, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	for _, s := range config.Servers {
		hydrateServerConfig(s)
	}
	return config, nil
}

func hydrateServerConfig(c *ServerConfig) {
	if c.RemoteServiceDirectory == "" {
		c.RemoteServiceDirectory = DefaultRemoteServiceDirectory
	}
}

func SaveConfig(path string, c *Config) error {
	configJson, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config file: %w", err)
	}
	if err = os.WriteFile(path, configJson, 0744); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}
