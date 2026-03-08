package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/mrshanahan/deploy-assets/pkg/config"

	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/service"
)

const (
	ServiceConfigFileName string = "config.json"
)

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

func (c *ServerConfig) LoadServiceConfig(exec config.Executor, name string) (*service.ServiceConfig, error) {
	servicePath, prs := c.Services[name]
	if !prs {
		return nil, fmt.Errorf("[%s] no service registered with name %s", exec.Name(), name)
	}

	serviceConfigPath := filepath.Join(servicePath, ServiceConfigFileName)
	if _, _, err := exec.ExecuteCommand("test", "-e", serviceConfigPath); err != nil {
		return nil, fmt.Errorf("[%s] failed to find config file for service at %s", exec.Name(), serviceConfigPath)
	}

	stdout, _, err := exec.ExecuteCommand("cat", serviceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("[%s] failed to get service config: %w", exec.Name(), err)
	}

	var serviceConfig *service.ServiceConfig
	if err := json.Unmarshal([]byte(stdout), &serviceConfig); err != nil {
		return nil, fmt.Errorf("[%s] failed to parse service config file: %w", exec.Name(), err)
	}

	return serviceConfig, nil
}

func (c *ServerConfig) SaveServiceConfig(exec config.Executor, name string, config *service.ServiceConfig) error {
	servicePath, prs := c.Services[name]
	if !prs {
		return fmt.Errorf("[%s] no service registered with name %s", exec.Name(), name)
	}

	serviceConfigPath := filepath.Join(servicePath, ServiceConfigFileName)

	configJson, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return fmt.Errorf("[%s] failed to serialize service config: %w", exec.Name(), err)
	}

	b64ConfigJson := base64.StdEncoding.EncodeToString(configJson)
	if _, _, err := exec.ExecuteShell(fmt.Sprintf("echo '%s' | base64 -d > '%s'", b64ConfigJson, serviceConfigPath)); err != nil {
		return fmt.Errorf("[%s] failed to save service config: %w", exec.Name(), err)
	}

	return nil
}
