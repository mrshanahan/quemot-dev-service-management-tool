package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/mrshanahan/deploy-assets/pkg/config"
)

const ()

type ServerConfig struct {
	Services map[string]string `json:"services"`
}

func LoadServerConfig(exec config.Executor, path string, force bool) (*ServerConfig, error) {
	if force {
		if _, _, err := exec.ExecuteShell(fmt.Sprintf("test -f '%s' || (echo '{}' > '%s')", path, path)); err != nil {
			return nil, fmt.Errorf("[%s] failed to check or create server config file %s: %w", exec.Name(), path, err)
		}
	}
	stdout, _, err := exec.ExecuteCommand("cat", path)
	if err != nil {
		return nil, fmt.Errorf("[%s] failed to read server config file %s: %w", exec.Name(), path, err)
	}
	var config *ServerConfig
	if err := json.Unmarshal([]byte(stdout), &config); err != nil {
		return nil, fmt.Errorf("[%s] failed to parse server config file %s: %w", exec.Name(), path, err)
	}
	return config, nil
}

func SaveServerConfig(exec config.Executor, path string, c *ServerConfig) error {
	configJson, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("[%s] failed to serialize server config file: %w", exec.Name(), err)
	}
	b64ConfigJson := base64.StdEncoding.EncodeToString([]byte(configJson))
	if _, _, err := exec.ExecuteShell(fmt.Sprintf("echo '%s' | base64 -d > '%s'", b64ConfigJson, path)); err != nil {
		return fmt.Errorf("[%s] failed to save server config: %w", exec.Name(), err)
	}

	return nil
}
