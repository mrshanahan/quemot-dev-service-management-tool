package service

type ServiceConfig struct {
	Commands map[string]string `json:"commands"`
}

func NewServiceConfig() *ServiceConfig {
	return &ServiceConfig{Commands: make(map[string]string)}
}
