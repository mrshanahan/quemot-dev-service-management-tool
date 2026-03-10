package service

import (
	"path/filepath"

	"github.com/mrshanahan/quemot-dev-service-management-tool/internal/install"
)

const (
	ServiceConfigFileName string = "config.json"
)

type ServiceDefinition struct {
	Path          string
	ServiceConfig *ServiceConfig
}

type ServiceConfig struct {
	Commands map[string]string `json:"commands"`
}

func NewServiceDefinition(name string) *ServiceDefinition {
	servicePath := filepath.Join(install.DefaultServicesDir, name)
	return &ServiceDefinition{
		Path:          servicePath,
		ServiceConfig: &ServiceConfig{Commands: make(map[string]string)},
	}
}

func GetDefaultConfigPath(servicePath string) string {
	return filepath.Join(servicePath, ServiceConfigFileName)
}
