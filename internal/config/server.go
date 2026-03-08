package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/mrshanahan/deploy-assets/pkg/config"
)

const ()

type ServerConfig struct {
	Services map[string]string `json:"services"`
}

func LoadRemoteServerConfig(exec config.Executor, path string, force bool) (*ServerConfig, error) {
	if force {
		if _, _, err := exec.ExecuteShell(fmt.Sprintf("test -f '%s' || (echo '{}' > '%s')", path, path)); err != nil {
			return nil, fmt.Errorf("failed to check or create remote config file %s: %w", path, err)
		}
	}
	stdout, _, err := exec.ExecuteCommand("cat", path)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote config file %s: %w", path, err)
	}
	var config *ServerConfig
	if err := json.Unmarshal([]byte(stdout), config); err != nil {
		return nil, fmt.Errorf("failed to parse remote config file %s: %w", path, err)
	}
	return config, nil
}

func SaveRemoteServerConfig(exec config.Executor, path string, c *ServerConfig) error {
	configJson, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize server config file: %w", err)
	}
	b64ConfigJson := base64.StdEncoding.EncodeToString([]byte(configJson))
	if _, _, err := exec.ExecuteShell(fmt.Sprintf("echo '%s' | base64 -d > '%s'", b64ConfigJson, path)); err != nil {
		return fmt.Errorf("failed to save remote server config: %w", err)
	}

	return nil
}

func LoadServerConfig(path string, force bool) (*ServerConfig, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !force {
				return nil, fmt.Errorf("server config file does not exist at %s: %w", path, err)
			}
			slog.Debug("server config file does not exist; creating", "path", path)
			defaultConfig := &ServerConfig{map[string]string{}}
			if err := SaveServerConfig(path, defaultConfig); err != nil {
				return nil, fmt.Errorf("failed to initialize server config file: %w", err)
			}
			return defaultConfig, nil
		}
		return nil, fmt.Errorf("failed to read server config file: %w", err)
	}

	var config *ServerConfig
	if err = json.Unmarshal(contents, &config); err != nil {
		return nil, fmt.Errorf("failed to parse server config file: %w", err)
	}

	return config, nil
}

func SaveServerConfig(path string, c *ServerConfig) error {
	configJson, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize server config file: %w", err)
	}
	if err = os.WriteFile(path, configJson, 0744); err != nil {
		return fmt.Errorf("failed to write server config file: %w", err)
	}
	return nil
}
